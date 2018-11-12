package hashtree

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	pathlib "path"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/OneOfOne/xxhash"
	bolt "github.com/coreos/bbolt"
	globlib "github.com/gobwas/glob"
	"github.com/pachyderm/pachyderm/src/client"
	"github.com/pachyderm/pachyderm/src/client/pfs"
	"github.com/pachyderm/pachyderm/src/client/pkg/pbutil"
	"github.com/pachyderm/pachyderm/src/server/pkg/errutil"
	"github.com/pachyderm/pachyderm/src/server/pkg/uuid"
)

const (
	FsBucket                = "fs"
	ChangedBucket           = "changed"
	ObjectBucket            = "object"
	DatumBucket             = "datum"
	perm                    = 0666
	DefaultBufSize          = 5 * (1 << (10 * 2))
	DefaultMergeConcurrency = 10
	IndexPath               = "-index"
	IndexSize               = uint64(1 << (10 * 2))
)

var (
	buckets   = []string{DatumBucket, FsBucket, ChangedBucket, ObjectBucket}
	exists    = []byte{1}
	nullByte  = []byte{0}
	slashByte = []byte{'/'}
	// A path should not have a globbing character
	SentinelByte = []byte{'*'}
)

func fs(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(b(FsBucket))
}

func changed(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(b(ChangedBucket))
}

func object(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(b(ObjectBucket))
}

func datum(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket(b(DatumBucket))
}

type dbHashTree struct {
	*bolt.DB
}

func slashEncode(b []byte) []byte {
	return bytes.Replace(b, slashByte, nullByte, -1)
}

func slashDecode(b []byte) []byte {
	return bytes.Replace(b, nullByte, slashByte, -1)
}

func s(b []byte) (result string) {
	if bytes.Equal(b, nullByte) {
		return ""
	}
	return string(slashDecode(b))
}

func b(s string) (result []byte) {
	if s == "" {
		return nullByte
	}
	return slashEncode([]byte(s))
}

func dbFile(storageRoot string) string {
	if storageRoot == "" {
		storageRoot = "/tmp"
	}
	return fmt.Sprintf("%s/hashtree/%s", storageRoot, uuid.NewWithoutDashes())
}

func NewDBHashTree(storageRoot string) (HashTree, error) {
	file := dbFile(storageRoot)
	if err := os.MkdirAll(pathlib.Dir(file), 0777); err != nil {
		return nil, err
	}
	result, err := newDBHashTree(file)
	if err != nil {
		return nil, err
	}
	if err := result.PutDir("/"); err != nil {
		return nil, err
	}
	return result, err
}

func DeserializeDBHashTree(storageRoot string, r io.Reader) (_ HashTree, retErr error) {
	result, err := NewDBHashTree(storageRoot)
	if err != nil {
		return nil, err
	}
	if err := result.Deserialize(r); err != nil {
		return nil, err
	}
	return result, nil
}

func newDBHashTree(file string) (HashTree, error) {
	db, err := bolt.Open(file, perm, nil)
	if err != nil {
		return nil, err
	}
	db.NoSync = true
	db.NoGrowSync = true
	db.MaxBatchDelay = 0
	if err := db.Batch(func(tx *bolt.Tx) error {
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists(b(bucket)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &dbHashTree{db}, nil
}

func get(tx *bolt.Tx, path string) (*NodeProto, error) {
	node := &NodeProto{}
	data := fs(tx).Get(b(path))
	if data == nil {
		return nil, errorf(PathNotFound, "file \"%s\" not found", path)
	}
	if err := node.Unmarshal(data); err != nil {
		return nil, err
	}
	return node, nil
}

func (h *dbHashTree) Get(path string) (*NodeProto, error) {
	path = clean(path)
	var node *NodeProto
	if err := h.View(func(tx *bolt.Tx) error {
		var err error
		node, err = get(tx, path)
		return err
	}); err != nil {
		return nil, err
	}
	return node, nil
}

func Get(rs []*Reader, filePath string) (*NodeProto, error) {
	filePath = clean(filePath)
	var fileNode *NodeProto
	if err := nodes(rs, func(path string, node *NodeProto) error {
		if path == filePath {
			fileNode = node
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if fileNode == nil {
		return nil, errorf(PathNotFound, "file \"%s\" not found", filePath)
	}
	return fileNode, nil
}

// dirPrefix returns the prefix that keys must have to be considered under the
// directory at path
func dirPrefix(path string) string {
	if path == "" {
		return "" // all paths are under the root
	}
	return path + "/"
}

// iterDir iterates through the nodes under path, it errors with PathNotFound if path doesn't exist, it errors with PathConflict if path exists but isn't a directory.
func iterDir(tx *bolt.Tx, path string, f func(k, v []byte, c *bolt.Cursor) error) error {
	node, err := get(tx, path)
	if err != nil {
		return err
	}
	if node.DirNode == nil {
		return errorf(PathConflict, "the file at \"%s\" is not a directory",
			path)
	}
	c := NewChildCursor(tx, path)
	for k, v := c.K(), c.V(); k != nil; k, v = c.Next() {
		if err := f(k, v, c.c); err != nil {
			if err == errutil.ErrBreak {
				return nil
			}
			return err
		}
	}
	return nil
}

func list(tx *bolt.Tx, path string, f func(*NodeProto) error) error {
	return iterDir(tx, path, func(_, v []byte, _ *bolt.Cursor) error {
		node := &NodeProto{}
		if err := node.Unmarshal(v); err != nil {
			return err
		}
		return f(node)
	})
}

// List retrieves the list of files and subdirectories of the directory at
// 'path'.
func (h *dbHashTree) List(path string, f func(*NodeProto) error) error {
	path = clean(path)
	return h.View(func(tx *bolt.Tx) error {
		return list(tx, path, f)
	})
}

func (h *dbHashTree) ListAll(path string) ([]*NodeProto, error) {
	var result []*NodeProto
	if err := h.List(path, func(node *NodeProto) error {
		result = append(result, node)
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func List(rs []*Reader, pattern string, f func(string, *NodeProto) error) (retErr error) {
	pattern = clean(pattern)
	if pattern == "" {
		pattern = "/"
	}
	g, err := globlib.Compile(pattern, '/')
	if err != nil {
		return errorf(MalformedGlob, err.Error())
	}
	return nodes(rs, func(path string, node *NodeProto) error {
		if (g.Match(path) && node.DirNode == nil) || (g.Match(pathlib.Dir(path))) {
			return f(path, node)
		}
		return nil
	})
}

func glob(tx *bolt.Tx, pattern string, f func(string, *NodeProto) error) error {
	if !IsGlob(pattern) {
		node, err := get(tx, pattern)
		if err != nil {
			return err
		}
		return f(externalDefault(pattern), node)
	}

	g, err := globlib.Compile(pattern, '/')
	if err != nil {
		return errorf(MalformedGlob, err.Error())
	}
	c := fs(tx).Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		if g.Match(s(k)) {
			node := &NodeProto{}
			if node.Unmarshal(v); err != nil {
				return err
			}
			if err := f(externalDefault(s(k)), node); err != nil {
				if err == errutil.ErrBreak {
					return nil
				}
				return err
			}
		}
	}
	return nil
}

func (h *dbHashTree) Glob(pattern string, f func(string, *NodeProto) error) error {
	pattern = clean(pattern)
	return h.View(func(tx *bolt.Tx) error {
		return glob(tx, pattern, f)
	})
}

func (h *dbHashTree) GlobAll(pattern string) (map[string]*NodeProto, error) {
	res := make(map[string]*NodeProto)
	if err := h.Glob(pattern, func(path string, node *NodeProto) error {
		res[path] = node
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}

func Glob(rs []*Reader, pattern string, f func(string, *NodeProto) error) (retErr error) {
	pattern = clean(pattern)
	g, err := globlib.Compile(pattern, '/')
	if err != nil {
		return errorf(MalformedGlob, err.Error())
	}
	return nodes(rs, func(path string, node *NodeProto) error {
		if g.Match(path) {
			return f(externalDefault(path), node)
		}
		return nil
	})
}

func (h *dbHashTree) FSSize() int64 {
	rootNode, err := h.Get("/")
	if err != nil {
		return 0
	}
	return rootNode.SubtreeSize
}

func (h *dbHashTree) Walk(path string, f func(path string, node *NodeProto) error) error {
	path = clean(path)
	return h.View(func(tx *bolt.Tx) error {
		c := fs(tx).Cursor()
		for k, v := c.Seek(b(path)); k != nil && strings.HasPrefix(s(k), path); k, v = c.Next() {
			node := &NodeProto{}
			if err := node.Unmarshal(v); err != nil {
				return err
			}
			nodePath := s(k)
			if nodePath == "" {
				nodePath = "/"
			}
			if nodePath != path && !strings.HasPrefix(nodePath, path+"/") {
				// node is a sibling of path, and thus doesn't get walked
				continue
			}
			if err := f(nodePath, node); err != nil {
				if err == errutil.ErrBreak {
					return nil
				}
				return err
			}
		}
		return nil
	})
}

func Walk(rs []*Reader, walkPath string, f func(path string, node *NodeProto) error) error {
	walkPath = clean(walkPath)
	return nodes(rs, func(path string, node *NodeProto) error {
		if path == "" {
			path = "/"
		}
		if path != walkPath && !strings.HasPrefix(path, walkPath+"/") {
			return nil
		}
		if err := f(path, node); err != nil {
			if err == errutil.ErrBreak {
				return nil
			}
			return err
		}
		return nil
	})
}

func diff(newTx, oldTx *bolt.Tx, newPath string, oldPath string, recursiveDepth int64, f func(string, *NodeProto, bool) error) error {
	newNode, err := get(newTx, clean(newPath))
	if err != nil && Code(err) != PathNotFound {
		return err
	}
	oldNode, err := get(oldTx, clean(oldPath))
	if err != nil && Code(err) != PathNotFound {
		return err
	}
	if (newNode == nil && oldNode == nil) ||
		(newNode != nil && oldNode != nil && bytes.Equal(newNode.Hash, oldNode.Hash)) {
		return nil
	}
	var newC *childCursor
	if newNode != nil {
		if newNode.FileNode != nil || recursiveDepth == 0 {
			if err := f(newPath, newNode, true); err != nil {
				return err
			}
		} else if newNode.DirNode != nil {
			newC = NewChildCursor(newTx, newPath)
		}
	}
	var oldC *childCursor
	if oldNode != nil {
		if oldNode.FileNode != nil || recursiveDepth == 0 {
			if err := f(oldPath, oldNode, false); err != nil {
				return err
			}
		} else if oldNode.DirNode != nil {
			oldC = NewChildCursor(oldTx, newPath)
		}
	}
	if recursiveDepth > 0 || recursiveDepth == -1 {
		newDepth := recursiveDepth
		if recursiveDepth > 0 {
			newDepth--
		}
		switch {
		case oldC == nil && newC == nil:
			return nil
		case oldC == nil:
			for k := newC.K(); k != nil; k, _ = newC.Next() {
				child := pathlib.Base(s(k))
				if err := diff(newTx, oldTx, pathlib.Join(newPath, child), pathlib.Join(oldPath, child), newDepth, f); err != nil {
					return err
				}
			}
		case newC == nil:
			for k := oldC.K(); k != nil; k, _ = oldC.Next() {
				child := pathlib.Base(s(k))
				if err := diff(newTx, oldTx, pathlib.Join(newPath, child), pathlib.Join(oldPath, child), newDepth, f); err != nil {
					return err
				}
			}
		default:
		Children:
			for {
				var child string
				switch compare(newC, oldC) {
				case -1:
					child = pathlib.Base(s(newC.K()))
					newC.Next()
				case 0:
					if len(newC.K()) == 0 {
						break Children
					}
					child = pathlib.Base(s(newC.K()))
					newC.Next()
					oldC.Next()
				case 1:
					child = pathlib.Base(s(oldC.K()))
					oldC.Next()
				}
				if err := diff(newTx, oldTx, pathlib.Join(newPath, child), pathlib.Join(oldPath, child), newDepth, f); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (h *dbHashTree) Diff(oldHashTree HashTree, newPath string, oldPath string, recursiveDepth int64, f func(path string, node *NodeProto, new bool) error) (retErr error) {
	// Setup a txn for each hashtree, this is a bit complicated because we don't want to make 2 read tx to the same tree, if we did then should someone start a write tx inbetween them we would have a deadlock
	old := oldHashTree.(*dbHashTree)
	if old == nil {
		return fmt.Errorf("unrecognized HashTree type")
	}
	rollback := func(tx *bolt.Tx) {
		if err := tx.Rollback(); err != nil && retErr == nil {
			retErr = err
		}
	}
	var newTx *bolt.Tx
	var oldTx *bolt.Tx
	if h == oldHashTree {
		tx, err := h.Begin(false)
		if err != nil {
			return err
		}
		newTx = tx
		oldTx = tx
		defer rollback(tx)
	} else {
		var err error
		newTx, err = h.Begin(false)
		if err != nil {
			return err
		}
		defer rollback(newTx)
		oldTx, err = old.Begin(false)
		if err != nil {
			return err
		}
		defer rollback(oldTx)
	}
	return diff(newTx, oldTx, newPath, oldPath, recursiveDepth, f)
}

func (h *dbHashTree) Serialize(_w io.Writer) error {
	w := pbutil.NewWriter(_w)
	return h.View(func(tx *bolt.Tx) error {
		for _, bucket := range buckets {
			b := tx.Bucket(b(bucket))
			if _, err := w.Write(
				&BucketHeader{
					Bucket: bucket,
				}); err != nil {
				return err
			}
			if err := b.ForEach(func(k, v []byte) error {
				if _, err := w.WriteBytes(k); err != nil {
					return err
				}
				_, err := w.WriteBytes(v)
				return err
			}); err != nil {
				return err
			}
			if _, err := w.WriteBytes(SentinelByte); err != nil {
				return err
			}
		}
		return nil
	})
}

type keyValue struct {
	k, v []byte
}

func (h *dbHashTree) Deserialize(_r io.Reader) error {
	r := pbutil.NewReader(_r)
	hdr := &BucketHeader{}
	batchSize := 10000
	kvs := make(chan *keyValue, batchSize/10)
	var eg errgroup.Group
	eg.Go(func() error {
		var bucket []byte
		for {
			count := 0
			if err := h.Update(func(tx *bolt.Tx) error {
				if bucket != nil {
					tx.Bucket(bucket).FillPercent = 1
				}
				for kv := range kvs {
					if kv.k == nil {
						bucket = kv.v
						continue
					}
					if err := tx.Bucket(bucket).Put(kv.k, kv.v); err != nil {
						return err
					}
					count++
					if count >= batchSize {
						return nil
					}
				}
				return nil
			}); err != nil {
				return err
			}
			if count <= 0 {
				return nil
			}
		}
	})
	for {
		hdr.Reset()
		if err := r.Read(hdr); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		bucket := b(hdr.Bucket)
		kvs <- &keyValue{nil, bucket}
		for {
			_k, err := r.ReadBytes()
			if err != nil {
				return err
			}
			if bytes.Equal(_k, SentinelByte) {
				break
			}
			// we need to make copies of k and v because the memory will be reused
			k := make([]byte, len(_k))
			copy(k, _k)
			_v, err := r.ReadBytes()
			if err != nil {
				return err
			}
			v := make([]byte, len(_v))
			copy(v, _v)
			kvs <- &keyValue{k, v}
		}
	}
	close(kvs)
	return eg.Wait()
}

func (h *dbHashTree) Copy() (HashTree, error) {
	if err := h.Hash(); err != nil {
		return nil, err
	}
	r, w := io.Pipe()
	var eg errgroup.Group
	eg.Go(func() (retErr error) {
		defer func() {
			if err := w.Close(); err != nil && retErr == nil {
				retErr = err
			}
		}()
		return h.Serialize(w)
	})
	var result HashTree
	eg.Go(func() error {
		var err error
		result, err = DeserializeDBHashTree(pathlib.Dir(h.Path()), r)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *dbHashTree) Destroy() error {
	path := h.Path()
	if err := h.Close(); err != nil {
		return err
	}
	return os.Remove(path)
}

func put(tx *bolt.Tx, path string, node *NodeProto) error {
	data, err := node.Marshal()
	if err != nil {
		return err
	}
	if err := changed(tx).Put(b(path), exists); err != nil {
		return fmt.Errorf("error putting \"%s\": %v", path, err)
	}
	return fs(tx).Put(b(path), data)
}

// visit visits every ancestor of 'path' (excluding 'path' itself), leaf to
// root (i.e.  end of 'path' to beginning), and calls 'update' on each node
// along the way. For example, if 'visit' is called with 'path'="/path/to/file",
// then updateFn is called as follows:
//
// 1. update(node at "/path/to" or nil, "/path/to", "file")
// 2. update(node at "/path"    or nil, "/path",    "to")
// 3. update(node at "/"        or nil, "",         "path")
//
// This is useful for propagating changes to size upwards.
func visit(tx *bolt.Tx, path string, update updateFn) error {
	for path != "" {
		parent, child := split(path)
		pnode, err := get(tx, parent)
		if err != nil && Code(err) != PathNotFound {
			return err
		}
		if pnode != nil && pnode.nodetype() != directory {
			return errorf(PathConflict, "attempted to visit \"%s\", but it's not a "+
				"directory", path)
		}
		if pnode == nil {
			pnode = &NodeProto{}
		}
		if err := update(pnode, parent, child); err != nil {
			return err
		}
		if err := put(tx, parent, pnode); err != nil {
			return err
		}
		path = parent
	}
	return nil
}

func (h *dbHashTree) PutFile(path string, objects []*pfs.Object, size int64) error {
	return h.putFile(path, objects, nil, size)
}

func (h *dbHashTree) PutFileOverwrite(path string, objects []*pfs.Object, overwriteIndex *pfs.OverwriteIndex, sizeDelta int64) error {
	return h.putFile(path, objects, overwriteIndex, sizeDelta)
}

func (h *dbHashTree) putFile(path string, objects []*pfs.Object, overwriteIndex *pfs.OverwriteIndex, sizeDelta int64) error {
	path = clean(path)
	return h.Batch(func(tx *bolt.Tx) error {
		node, err := get(tx, path)
		if err != nil && Code(err) != PathNotFound {
			return err
		}
		if node == nil {
			node = &NodeProto{
				Name:     base(path),
				FileNode: &FileNodeProto{},
			}
		} else if node.nodetype() != file {
			return errorf(PathConflict, "could not put file at \"%s\"; a file of "+
				"type %s is already there", path, node.nodetype().tostring())
		}
		// Append new objects.  Remove existing objects if overwriting.
		if overwriteIndex != nil && overwriteIndex.Index <= int64(len(node.FileNode.Objects)) {
			node.FileNode.Objects = node.FileNode.Objects[:overwriteIndex.Index]
		}
		node.SubtreeSize += sizeDelta
		node.FileNode.Objects = append(node.FileNode.Objects, objects...)
		// Put the node
		if err := put(tx, path, node); err != nil {
			return err
		}
		return visit(tx, path, func(node *NodeProto, parent, child string) error {
			if node.DirNode == nil {
				// node created as part of this visit call, fill in the basics
				node.Name = base(parent)
				node.DirNode = &DirectoryNodeProto{}
			}
			node.SubtreeSize += sizeDelta
			return nil
		})
	})
}

func (h *dbHashTree) PutFileBlockRefs(path string, blockRefs []*pfs.BlockRef, size int64) error {
	return h.putFileBlockRefs(path, blockRefs, size)
}

func (h *dbHashTree) putFileBlockRefs(path string, blockRefs []*pfs.BlockRef, sizeDelta int64) error {
	path = clean(path)
	return h.Batch(func(tx *bolt.Tx) error {
		node, err := get(tx, path)
		if err != nil && Code(err) != PathNotFound {
			return err
		}
		if node == nil {
			node = &NodeProto{
				Name:     base(path),
				FileNode: &FileNodeProto{},
			}
		} else if node.nodetype() != file {
			return errorf(PathConflict, "could not put file at \"%s\"; a file of "+
				"type %s is already there", path, node.nodetype().tostring())
		}
		node.SubtreeSize += sizeDelta
		node.FileNode.BlockRefs = append(node.FileNode.BlockRefs, blockRefs...)
		// Put the node
		if err := put(tx, path, node); err != nil {
			return err
		}
		return visit(tx, path, func(node *NodeProto, parent, child string) error {
			if node.DirNode == nil {
				// node created as part of this visit call, fill in the basics
				node.Name = base(parent)
				node.DirNode = &DirectoryNodeProto{}
			}
			node.SubtreeSize += sizeDelta
			return nil
		})
	})
}

func (h *dbHashTree) PutDir(path string) error {
	path = clean(path)
	return h.Batch(func(tx *bolt.Tx) error {
		node, err := get(tx, path)
		if err != nil && Code(err) != PathNotFound {
			return err
		}
		if node != nil {
			if node.nodetype() == directory {
				return nil
			} else if node.nodetype() != none {
				return errorf(PathConflict, "could not create directory at \"%s\"; a "+
					"file of type %s is already there", path, node.nodetype().tostring())
			}
		}
		node = &NodeProto{
			Name:    base(path),
			DirNode: &DirectoryNodeProto{},
		}
		if err := put(tx, path, node); err != nil {
			return err
		}
		return visit(tx, path, func(node *NodeProto, parent, child string) error {
			if node.DirNode == nil {
				// node created as part of this visit call, fill in the basics
				node.Name = base(parent)
				node.DirNode = &DirectoryNodeProto{}
			}
			return nil
		})
	})
}

// deleteDir deletes a directory and all the children under it
func deleteDir(tx *bolt.Tx, path string) error {
	c := fs(tx).Cursor()
	prefix := append(b(path), nullByte[0])
	for k, _ := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = c.Next() {
		if err := c.Delete(); err != nil {
			return err
		}
	}
	return fs(tx).Delete(b(path))
}

func (h *dbHashTree) DeleteFile(path string) error {
	path = clean(path)

	// Delete root means delete all files
	if path == "" {
		path = "/*"
	}
	return h.Batch(func(tx *bolt.Tx) error {
		if err := glob(tx, path, func(path string, node *NodeProto) error {
			// Check if the file has been deleted already
			if _, err := get(tx, path); err != nil && Code(err) == PathNotFound {
				return nil
			}
			// Remove 'path' and all nodes underneath it from h.fs
			if err := deleteDir(tx, path); err != nil {
				return err
			}
			size := node.SubtreeSize
			// Remove 'path' from its parent directory
			// TODO(bryce) Decide if this should be removed.
			parent, _ := split(path)
			pnode, err := get(tx, parent)
			if err != nil {
				if Code(err) == PathNotFound {
					return errorf(Internal, "delete discovered orphaned file \"%s\"", path)
				}
				return err
			}
			if pnode.DirNode == nil {
				return errorf(Internal, "file at \"%s\" is a regular-file, but \"%s\" already exists "+
					"under it (likely an uncaught PathConflict in prior PutFile or Merge)", path, pnode.DirNode)
			}
			put(tx, parent, pnode)
			// Mark nodes as 'changed' back to root
			if err := visit(tx, path, func(node *NodeProto, parent, child string) error {
				// If node.DirNode is nil it means either the parent didn't
				// exist (and thus was deserialized fron nil) or it does exist
				// but thinks it's a file, both are errors.
				if node.DirNode == nil {
					return errorf(Internal,
						"encountered orphaned file \"%s\" while deleting \"%s\"", path,
						join(parent, child))
				}
				node.SubtreeSize -= size
				return nil
			}); err != nil {
				return err
			}
			return nil
		}); err != nil && Code(err) != PathNotFound {
			// Deleting a non-existent file should be a no-op
			return err
		}
		return nil
	})
}

type mergeNode struct {
	k, v      []byte
	nodeProto *NodeProto
}

type Reader struct {
	pbr    pbutil.Reader
	filter func(k []byte) (bool, error)
}

func NewReader(r io.Reader, filter func(k []byte) (bool, error)) *Reader {
	return &Reader{
		pbr:    pbutil.NewReader(r),
		filter: filter,
	}
}

func (r *Reader) Read() (*mergeNode, error) {
	_k, err := r.pbr.ReadBytes()
	if err != nil {
		return nil, err
	}
	if r.filter != nil {
		for {
			ok, err := r.filter(_k)
			if err != nil {
				return nil, err
			}
			if ok {
				break
			}
			_, err = r.pbr.ReadBytes()
			if err != nil {
				return nil, err
			}
			_k, err = r.pbr.ReadBytes()
			if err != nil {
				return nil, err
			}
		}

	}
	k := make([]byte, len(_k))
	copy(k, _k)
	_v, err := r.pbr.ReadBytes()
	if err != nil {
		return nil, err
	}
	v := make([]byte, len(_v))
	copy(v, _v)
	return &mergeNode{
		k: k,
		v: v,
	}, nil
}

type Writer struct {
	pbw    pbutil.Writer
	size   uint64
	idxs   []*Index
	offset uint64
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		pbw: pbutil.NewWriter(w),
	}
}

func (w *Writer) Write(n *mergeNode) error {
	// Marshal node if it was merged
	if n.nodeProto != nil {
		var err error
		n.v, err = n.nodeProto.Marshal()
		if err != nil {
			return err
		}
	}
	// Get size info from root node
	if bytes.Equal(n.k, nullByte) {
		if n.nodeProto == nil {
			n.nodeProto = &NodeProto{}
			if err := n.nodeProto.Unmarshal(n.v); err != nil {
				return err
			}
		}
		w.size = uint64(n.nodeProto.SubtreeSize)
	}
	// Write index for every index size bytes
	if w.offset > uint64(len(w.idxs)+1)*IndexSize {
		w.idxs = append(w.idxs, &Index{
			K:      n.k,
			Offset: w.offset,
		})
	}
	b, err := w.pbw.WriteBytes(n.k)
	if err != nil {
		return err
	}
	w.offset += uint64(b)
	b, err = w.pbw.WriteBytes(n.v)
	if err != nil {
		return err
	}
	w.offset += uint64(b)
	return nil
}

func (w *Writer) Copy(r *Reader) error {
	for {
		n, err := r.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := w.Write(n); err != nil {
			return err
		}
	}
}

func (w *Writer) Index() ([]byte, error) {
	buf := &bytes.Buffer{}
	pbw := pbutil.NewWriter(buf)
	for _, idx := range w.idxs {
		if _, err := pbw.Write(idx); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func GetRangeFromIndex(r io.Reader, prefix string) (uint64, uint64, error) {
	prefix = clean(prefix)
	pbr := pbutil.NewReader(r)
	idx := &Index{}
	k := b(prefix)
	var lower, upper uint64
	iter := func(f func(int) bool) error {
		for {
			if err := pbr.Read(idx); err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			var cmp int
			if len(k) < len(idx.K) {
				cmp = bytes.Compare(k, idx.K[:len(k)])
			} else {
				cmp = bytes.Compare(k[:len(idx.K)], idx.K)
			}
			if f(cmp) {
				break
			}
		}
		return nil
	}
	low := func(cmp int) bool {
		if cmp > 0 {
			lower = idx.Offset
			return false
		} else if cmp < 0 {
			// Handles the case where a prefix fits within one range
			upper = idx.Offset
		}
		return true
	}
	up := func(cmp int) bool {
		if cmp < 0 {
			upper = idx.Offset
			return true
		}
		return false
	}
	// Find lower
	iter(low)
	// Find upper
	if upper <= 0 {
		iter(up)
	}
	// Handles the case when at the end of the indexes
	if upper <= 0 {
		return lower, 0, nil
	}
	// Return offset and size
	return lower, upper - lower, nil
}

func NewFilter(numTrees int64, tree int64) func(k []byte) (bool, error) {
	return func(k []byte) (bool, error) {
		if pathToTree(k, numTrees) == uint64(tree) {
			return true, nil
		}
		return false, nil
	}
}

func PathToTree(path string, numTrees int64) uint64 {
	path = clean(path)
	return pathToTree(b(path), numTrees)
}

func pathToTree(k []byte, numTrees int64) uint64 {
	return xxhash.Checksum64(k) % uint64(numTrees)
}

type nodeStream struct {
	node *mergeNode
	r    *Reader
}

type mergePQ struct {
	q    []*nodeStream
	size int
}

func (mq *mergePQ) k(i int) []byte {
	return mq.q[i].node.k
}

func (mq *mergePQ) insert(s *nodeStream) error {
	// Get next node in stream
	var err error
	s.node, err = s.r.Read()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	mq.q[mq.size+1] = s
	mq.size++
	// Propagate insert up the queue
	i := mq.size
	for i > 1 {
		if bytes.Compare(mq.k(i/2), mq.k(i)) <= 0 {
			break
		}
		mq.swap(i/2, i)
		i /= 2
	}
	return nil
}

func (mq *mergePQ) next() ([]*mergeNode, error) {
	ns := []*mergeNode{mq.q[1].node}
	if err := mq.fill(); err != nil {
		return nil, err
	}
	// Keep popping nodes off the queue if they share the same path
	for mq.q[1] != nil && bytes.Compare(mq.k(1), ns[0].k) == 0 {
		ns = append(ns, mq.q[1].node)
		if err := mq.fill(); err != nil {
			return nil, err
		}
	}
	return ns, nil
}

func merge(ns []*mergeNode) (*mergeNode, error) {
	base := ns[0]
	if base.nodeProto == nil {
		base.nodeProto = &NodeProto{}
		if err := base.nodeProto.Unmarshal(base.v); err != nil {
			return nil, err
		}
	}
	for i := 1; i < len(ns); i++ {
		n := ns[i]
		n.nodeProto = &NodeProto{}
		if err := n.nodeProto.Unmarshal(n.v); err != nil {
			return nil, err
		}
		// Check for inconsistent node types
		if base.nodeProto.nodetype() != n.nodeProto.nodetype() {
			return nil, errorf(PathConflict, "could not merge path \"%s\" "+
				"which is a different type in different hashtrees", s(base.k))
		}
		// Merge file content
		if base.nodeProto.nodetype() == file {
			base.nodeProto.FileNode.BlockRefs = append(base.nodeProto.FileNode.BlockRefs, n.nodeProto.FileNode.BlockRefs...)

		}
		base.nodeProto.SubtreeSize += n.nodeProto.SubtreeSize
	}
	return base, nil
}

func (mq *mergePQ) fill() error {
	// Save stream for re-insert
	ns := mq.q[1]
	// Replace first stream with last
	mq.q[1] = mq.q[mq.size]
	mq.q[mq.size] = nil
	mq.size--
	// Propagate last stream down the queue
	i := 1
	var next int
	for {
		l, r := i*2, i*2+1
		if l > mq.size {
			break
		} else if r > mq.size || bytes.Compare(mq.k(l), mq.k(r)) <= 0 {
			next = l
		} else {
			next = r
		}
		if bytes.Compare(mq.k(i), mq.k(next)) <= 0 {
			break
		}
		mq.swap(i, next)
		i = next
	}
	// Re-insert stream
	return mq.insert(ns)
}

func (mq *mergePQ) swap(i, j int) {
	tmp := mq.q[i]
	mq.q[i] = mq.q[j]
	mq.q[j] = tmp
}

func Merge(w *Writer, rs []*Reader) (uint64, error) {
	mq := &mergePQ{q: make([]*nodeStream, len(rs)+1)}
	// Setup first set of nodes
	for _, r := range rs {
		if err := mq.insert(&nodeStream{r: r}); err != nil {
			return 0, err
		}
	}
	for mq.q[1] != nil {
		// Get next nodes to merge
		ns, err := mq.next()
		if err != nil {
			return 0, err
		}
		// Merge nodes
		n, err := merge(ns)
		if err != nil {
			return 0, err
		}
		// Write out result
		if err := w.Write(n); err != nil {
			return 0, err
		}
	}
	return w.size, nil
}

func nodes(rs []*Reader, f func(path string, nodeProto *NodeProto) error) error {
	mq := &mergePQ{q: make([]*nodeStream, len(rs)+1)}
	// Setup first set of nodes
	for _, r := range rs {
		if err := mq.insert(&nodeStream{r: r}); err != nil {
			return err
		}
	}
	for mq.q[1] != nil {
		// Get next node
		ns, err := mq.next()
		if err != nil {
			return err
		}
		// Unmarshal node and run callback
		n := ns[0]
		n.nodeProto = &NodeProto{}
		if err := n.nodeProto.Unmarshal(n.v); err != nil {
			return err
		}
		if err := f(s(n.k), n.nodeProto); err != nil {
			return err
		}
	}
	return nil
}

func (h *dbHashTree) PutObject(o *pfs.Object, blockRef *pfs.BlockRef) error {
	data, err := blockRef.Marshal()
	if err != nil {
		return err
	}
	return h.Batch(func(tx *bolt.Tx) error {
		return object(tx).Put(b(o.Hash), data)
	})
}

func (h *dbHashTree) GetObject(o *pfs.Object) (*pfs.BlockRef, error) {
	result := &pfs.BlockRef{}
	if err := h.View(func(tx *bolt.Tx) error {
		data := object(tx).Get(b(o.Hash))
		if data == nil {
			return errorf(ObjectNotFound, "object \"%s\" not found", o.Hash)
		}
		return result.Unmarshal(data)
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *dbHashTree) PutDatum(d string) error {
	return h.Batch(func(tx *bolt.Tx) error {
		return datum(tx).Put(b(d), exists)
	})
}

func (h *dbHashTree) HasDatum(d string) (bool, error) {
	var result bool
	if err := h.View(func(tx *bolt.Tx) error {
		val := datum(tx).Get(b(d))
		if val != nil {
			result = true
		}
		return nil
	}); err != nil {
		return false, err
	}
	return result, nil
}

func (h *dbHashTree) DiffDatums(datums chan string) (bool, error) {
	var result bool
	if err := h.Update(func(tx *bolt.Tx) error {
		for d := range datums {
			val := datum(tx).Get(b(d))
			if val != nil {
				if err := datum(tx).Put(b(d), nullByte); err != nil {
					return err
				}
			}
		}
		if err := datum(tx).ForEach(func(k, v []byte) error {
			if string(v) != string(nullByte) {
				result = true
			}
			return datum(tx).Put(k, exists)
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false, err
	}
	return result, nil
}

func hasChanged(tx *bolt.Tx, path string) bool {
	return changed(tx).Get(b(path)) != nil
}

func canonicalize(tx *bolt.Tx, path string) error {
	path = clean(path)
	if !hasChanged(tx, path) {
		return nil // Node is already canonical
	}
	n, err := get(tx, path)
	if err != nil {
		if Code(err) == PathNotFound {
			return errorf(Internal, "file \"%s\" not found; cannot canonicalize", path)
		}
		return err
	}

	// Compute hash of 'n'
	hash := sha256.New()
	switch n.nodetype() {
	case directory:
		// Compute n.Hash by concatenating name + hash of all children of n.DirNode
		// Note that the order of the children of n.DirNode are sorted when iterating.
		if err := iterDir(tx, path, func(k, _ []byte, _ *bolt.Cursor) error {
			childPath := s(k)
			if err := canonicalize(tx, childPath); err != nil {
				return err
			}
			childNode, err := get(tx, childPath)
			if err != nil {
				if Code(err) == PathNotFound {
					return errorf(Internal, "could not find file for \"%s\" while "+
						"updating hash of \"%s\"", childPath, path)
				}
				return err
			}
			// append child.Name and child.Hash to b
			hash.Write([]byte(fmt.Sprintf("%s:%s:", childNode.Name, childNode.Hash)))
			return nil
		}); err != nil {
			return err
		}
	case file:
		// Compute n.Hash by concatenating all BlockRef hashes in n.FileNode.
		for _, object := range n.FileNode.Objects {
			hash.Write([]byte(object.Hash))
		}
	default:
		return errorf(Internal,
			"malformed file at \"%s\" is neither a file nor a directory", path)
	}

	// Update hash of 'n'
	n.Hash = hash.Sum(nil)
	if err := put(tx, path, n); err != nil {
		return err
	}
	return changed(tx).Delete(b(path))
}

func (h *dbHashTree) Hash() error {
	return h.Batch(func(tx *bolt.Tx) error {
		return canonicalize(tx, "")
	})
}

type nodetype uint8

const (
	none         nodetype = iota // No file is present at this point in the tree
	directory                    // The file at this point in the tree is a directory
	file                         // ... is a regular file
	unrecognized                 // ... is an an unknown type
)

func (n *NodeProto) nodetype() nodetype {
	switch {
	case n == nil || (n.DirNode == nil && n.FileNode == nil):
		return none
	case n.DirNode != nil:
		return directory
	case n.FileNode != nil:
		return file
	default:
		return unrecognized
	}
}

func (n nodetype) tostring() string {
	switch n {
	case none:
		return "none"
	case directory:
		return "directory"
	case file:
		return "file"
	default:
		return "unknown"
	}
}

// updateFn is used by 'visit'. The first parameter is the node being visited,
// the second parameter is the path of that node, and the third parameter is the
// child of that node from the 'path' argument to 'visit'.
//
// The *NodeProto argument is guaranteed to have DirNode set (if it's not nil)--visit
// returns a 'PathConflict' error otherwise.
type updateFn func(*NodeProto, string, string) error

// This can be passed to visit() to detect PathConflict errors early
func nop(*NodeProto, string, string) error {
	return nil
}

var globRegex = regexp.MustCompile(`[*?\[\]\{\}!]`)

// IsGlob checks if the pattern contains a glob character
func IsGlob(pattern string) bool {
	pattern = clean(pattern)
	return globRegex.Match([]byte(pattern))
}

// GlobLiteralPrefix returns the prefix before the first glob character
func GlobLiteralPrefix(pattern string) string {
	pattern = clean(pattern)
	idx := globRegex.FindStringIndex(pattern)
	if idx == nil {
		return pattern
	}
	return pattern[:idx[0]]
}

// GetHashTreeObject is a convenience function to deserialize a HashTree from an object in the object store.
func GetHashTreeObject(pachClient *client.APIClient, storageRoot string, treeRef *pfs.Object) (HashTree, error) {
	return getHashTree(storageRoot, func(w io.Writer) error {
		return pachClient.GetObject(treeRef.Hash, w)
	})
}

// GetHashTreeObject is a convenience function to deserialize a HashTree from an tagged object in the object store.
func GetHashTreeTag(pachClient *client.APIClient, storageRoot string, treeRef *pfs.Tag) (HashTree, error) {
	return getHashTree(storageRoot, func(w io.Writer) error {
		return pachClient.GetTag(treeRef.Name, w)
	})
}

func getHashTree(storageRoot string, f func(io.Writer) error) (_ HashTree, retErr error) {
	filePath := dbFile(storageRoot)
	if err := os.MkdirAll(pathlib.Dir(filePath), 0777); err != nil {
		return nil, err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil && retErr == nil {
			retErr = err
		}
		if err := os.Remove(filePath); err != nil && retErr == nil {
			retErr = err
		}
	}()
	if err := f(file); err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	return DeserializeDBHashTree(storageRoot, file)
}

// PutHashTree is a convenience function for putting a HashTree to an object store.
func PutHashTree(pachClient *client.APIClient, tree HashTree, tags ...string) (*pfs.Object, error) {
	r, w := io.Pipe()
	var eg errgroup.Group
	eg.Go(func() (retErr error) {
		defer func() {
			if err := w.Close(); err != nil && retErr == nil {
				retErr = err
			}
		}()
		return tree.Serialize(w)
	})
	var treeRef *pfs.Object
	eg.Go(func() error {
		var err error
		treeRef, _, err = pachClient.PutObject(r, tags...)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return treeRef, nil
}

// childCursor efficiently iterates the children of a directory
type childCursor struct {
	c *bolt.Cursor
	// childCursor efficiently iterates the children of a directory
	dir []byte
	k   []byte
	v   []byte
}

func NewChildCursor(tx *bolt.Tx, path string) *childCursor {
	path = clean(path)
	c := fs(tx).Cursor()
	dir := b(path)
	k, v := c.Seek(append(dir, nullByte[0]))
	if !bytes.Equal(dir, nullByte) {
		dir = append(dir, nullByte[0])
	}
	if !bytes.HasPrefix(k, dir) {
		k, v = nil, nil
	}
	return &childCursor{
		c:   c,
		dir: dir,
		k:   k,
		v:   v,
	}
}

func (d *childCursor) K() []byte {
	return d.k
}

func (d *childCursor) V() []byte {
	return d.v
}

func (d *childCursor) Next() ([]byte, []byte) {
	if d.k == nil {
		return nil, nil
	}
	k, v := d.c.Seek(append(d.k, 1))
	if !bytes.HasPrefix(k, d.dir) {
		k, v = nil, nil
	}
	d.k, d.v = k, v
	return k, v
}

func compare(a, b *childCursor) int {
	switch {
	case a.k == nil && b.k == nil:
		return 0
	case b.k == nil:
		return -1
	case a.k == nil:
		return 1
	default:
		return bytes.Compare(bytes.TrimPrefix(a.k, a.dir), bytes.TrimPrefix(b.k, b.dir))
	}
}

// Ordered is an in memory version of the hashtree that is optimized and only works for lexicographically ordered inserts followed by serialization.
type Ordered struct {
	fs       []*node
	dirStack []*node
	root     string
}

type node struct {
	path      string
	nodeProto *NodeProto
	hash      hash.Hash
}

func NewOrdered(root string) *Ordered {
	root = clean(root)
	o := &Ordered{}
	n := &node{
		path: "",
		nodeProto: &NodeProto{
			Name:    "",
			DirNode: &DirectoryNodeProto{},
		},
		hash: sha256.New(),
	}
	o.fs = append(o.fs, n)
	o.dirStack = append(o.dirStack, n)
	o.mkdirAll(root)
	o.root = root
	return o
}

func (o *Ordered) mkdirAll(path string) {
	var paths []string
	for path != "" {
		paths = append(paths, path)
		path, _ = split(path)
	}
	for i := len(paths) - 1; i >= 0; i-- {
		o.PutDir(paths[i])
	}
}

func (o *Ordered) PutDir(path string) {
	path = clean(path)
	if path == "" {
		return
	}
	nodeProto := &NodeProto{
		Name:    base(path),
		DirNode: &DirectoryNodeProto{},
	}
	o.putDir(path, nodeProto)
}

func (o *Ordered) putDir(path string, nodeProto *NodeProto) {
	path = join(o.root, path)
	o.handleEndOfDirectory(path)
	n := &node{
		path:      path,
		nodeProto: nodeProto,
		hash:      sha256.New(),
	}
	o.fs = append(o.fs, n)
	o.dirStack = append(o.dirStack, n)
}

func (o *Ordered) PutFile(path string, hash []byte, size int64, fileNodeProto *FileNodeProto) {
	path = clean(path)
	nodeProto := &NodeProto{
		Name:        base(path),
		Hash:        hash,
		SubtreeSize: size,
		FileNode:    fileNodeProto,
	}
	o.putFile(path, nodeProto)
}

func (o *Ordered) putFile(path string, nodeProto *NodeProto) {
	path = join(o.root, path)
	o.handleEndOfDirectory(path)
	n := &node{
		path:      path,
		nodeProto: nodeProto,
	}
	o.fs = append(o.fs, n)
	o.dirStack[len(o.dirStack)-1].hash.Write([]byte(fmt.Sprintf("%s:%s:", n.nodeProto.Name, n.nodeProto.Hash)))
	o.dirStack[len(o.dirStack)-1].nodeProto.SubtreeSize += nodeProto.SubtreeSize
}

func (o *Ordered) handleEndOfDirectory(path string) {
	parent, _ := split(path)
	if parent != o.dirStack[len(o.dirStack)-1].path {
		child := o.dirStack[len(o.dirStack)-1]
		child.nodeProto.Hash = child.hash.Sum(nil)
		o.dirStack = o.dirStack[:len(o.dirStack)-1]
		parent := o.dirStack[len(o.dirStack)-1]
		parent.hash.Write([]byte(fmt.Sprintf("%s:%s:", child.nodeProto.Name, child.nodeProto.Hash)))
		parent.nodeProto.SubtreeSize += child.nodeProto.SubtreeSize
	}
}

func (o *Ordered) Serialize(_w io.Writer) error {
	w := NewWriter(_w)
	o.fs[0].nodeProto.Hash = o.fs[0].hash.Sum(nil)
	for _, n := range o.fs {
		if err := w.Write(&mergeNode{
			k:         b(n.path),
			nodeProto: n.nodeProto,
		}); err != nil {
			return err
		}
	}
	return nil
}

type Unordered struct {
	fs   map[string]*NodeProto
	root string
}

func NewUnordered(root string) *Unordered {
	return &Unordered{
		fs:   make(map[string]*NodeProto),
		root: clean(root),
	}
}

func (u *Unordered) PutFile(path string, hash []byte, size int64, blockRefs ...*pfs.BlockRef) {
	path = join(u.root, path)
	nodeProto := &NodeProto{
		Name:        base(path),
		Hash:        hash,
		SubtreeSize: size,
		FileNode: &FileNodeProto{
			BlockRefs: blockRefs,
		},
	}
	u.fs[path] = nodeProto
	u.createParents(path)
}

func (u *Unordered) createParents(path string) {
	if path != "" {
		path, _ = split(path)
		path = clean(path)
		if _, ok := u.fs[path]; ok {
			return
		}
		nodeProto := &NodeProto{
			Name:    base(path),
			DirNode: &DirectoryNodeProto{},
		}
		u.fs[path] = nodeProto
		u.createParents(path)
	}
}

func (u *Unordered) Ordered() *Ordered {
	paths := make([]string, len(u.fs))
	i := 0
	for path := range u.fs {
		paths[i] = path
		i++
	}
	sort.Strings(paths)
	o := NewOrdered("")
	for _, path := range paths {
		n := u.fs[path]
		if n.DirNode != nil {
			o.putDir(path, n)
		} else {
			o.putFile(path, n)
		}
	}
	return o
}
