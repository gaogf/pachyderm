// Code generated by protoc-gen-go.
// source: pfs.proto
// DO NOT EDIT!

/*
Package pfs is a generated protocol buffer package.

It is generated from these files:
	pfs.proto

It has these top-level messages:
	Repository
	Commit
	Path
	Shard
	CommitInfo
	InitRepositoryRequest
	InitRepositoryResponse
	GetFileRequest
	MakeDirectoryRequest
	MakeDirectoryResponse
	PutFileRequest
	PutFileResponse
	ListFilesRequest
	ListFilesResponse
	GetParentRequest
	GetParentResponse
	BranchRequest
	BranchResponse
	CommitRequest
	CommitResponse
	PullDiffRequest
	PushDiffRequest
	PushDiffResponse
	GetCommitInfoRequest
	GetCommitInfoResponse
*/
package pfs

import proto "github.com/golang/protobuf/proto"
import google_protobuf "github.com/peter-edge/go-google-protobuf"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal

// DriverType represents the driver type used by the implementation of PFS.
type DriverType int32

const (
	DriverType_DRIVER_TYPE_NONE  DriverType = 0
	DriverType_DRIVER_TYPE_BTRFS DriverType = 1
)

var DriverType_name = map[int32]string{
	0: "DRIVER_TYPE_NONE",
	1: "DRIVER_TYPE_BTRFS",
}
var DriverType_value = map[string]int32{
	"DRIVER_TYPE_NONE":  0,
	"DRIVER_TYPE_BTRFS": 1,
}

func (x DriverType) String() string {
	return proto.EnumName(DriverType_name, int32(x))
}

// CommitType represents the type of commit.
type CommitType int32

const (
	CommitType_COMMIT_TYPE_NONE  CommitType = 0
	CommitType_COMMIT_TYPE_READ  CommitType = 1
	CommitType_COMMIT_TYPE_WRITE CommitType = 2
)

var CommitType_name = map[int32]string{
	0: "COMMIT_TYPE_NONE",
	1: "COMMIT_TYPE_READ",
	2: "COMMIT_TYPE_WRITE",
}
var CommitType_value = map[string]int32{
	"COMMIT_TYPE_NONE":  0,
	"COMMIT_TYPE_READ":  1,
	"COMMIT_TYPE_WRITE": 2,
}

func (x CommitType) String() string {
	return proto.EnumName(CommitType_name, int32(x))
}

// WriteCommitType represents the type for a write commit.
type WriteCommitType int32

const (
	WriteCommitType_WRITE_COMMIT_TYPE_NONE WriteCommitType = 0
	WriteCommitType_WRITE_COMMIT_TYPE_PUT  WriteCommitType = 1
	WriteCommitType_WRITE_COMMIT_TYPE_PUSH WriteCommitType = 2
)

var WriteCommitType_name = map[int32]string{
	0: "WRITE_COMMIT_TYPE_NONE",
	1: "WRITE_COMMIT_TYPE_PUT",
	2: "WRITE_COMMIT_TYPE_PUSH",
}
var WriteCommitType_value = map[string]int32{
	"WRITE_COMMIT_TYPE_NONE": 0,
	"WRITE_COMMIT_TYPE_PUT":  1,
	"WRITE_COMMIT_TYPE_PUSH": 2,
}

func (x WriteCommitType) String() string {
	return proto.EnumName(WriteCommitType_name, int32(x))
}

// Repository represents a repository.
type Repository struct {
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
}

func (m *Repository) Reset()         { *m = Repository{} }
func (m *Repository) String() string { return proto.CompactTextString(m) }
func (*Repository) ProtoMessage()    {}

// Commit represents a specific commit in a repository.
type Commit struct {
	Repository *Repository `protobuf:"bytes,1,opt,name=repository" json:"repository,omitempty"`
	Id         string      `protobuf:"bytes,2,opt,name=id" json:"id,omitempty"`
}

func (m *Commit) Reset()         { *m = Commit{} }
func (m *Commit) String() string { return proto.CompactTextString(m) }
func (*Commit) ProtoMessage()    {}

func (m *Commit) GetRepository() *Repository {
	if m != nil {
		return m.Repository
	}
	return nil
}

// Path represents the full path to a file or directory within PFS.
type Path struct {
	Commit *Commit `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
	Path   string  `protobuf:"bytes,2,opt,name=path" json:"path,omitempty"`
}

func (m *Path) Reset()         { *m = Path{} }
func (m *Path) String() string { return proto.CompactTextString(m) }
func (*Path) ProtoMessage()    {}

func (m *Path) GetCommit() *Commit {
	if m != nil {
		return m.Commit
	}
	return nil
}

// Shard represents a dynamic shard within PFS.
// number must alway be less than modulo.
type Shard struct {
	Number uint64 `protobuf:"varint,1,opt,name=number" json:"number,omitempty"`
	Modulo uint64 `protobuf:"varint,2,opt,name=modulo" json:"modulo,omitempty"`
}

func (m *Shard) Reset()         { *m = Shard{} }
func (m *Shard) String() string { return proto.CompactTextString(m) }
func (*Shard) ProtoMessage()    {}

// CommitInfo represents information about a commit.
type CommitInfo struct {
	Commit     *Commit    `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
	CommitType CommitType `protobuf:"varint,2,opt,name=commit_type,enum=pfs.CommitType" json:"commit_type,omitempty"`
}

func (m *CommitInfo) Reset()         { *m = CommitInfo{} }
func (m *CommitInfo) String() string { return proto.CompactTextString(m) }
func (*CommitInfo) ProtoMessage()    {}

func (m *CommitInfo) GetCommit() *Commit {
	if m != nil {
		return m.Commit
	}
	return nil
}

type InitRepositoryRequest struct {
	Repository *Repository `protobuf:"bytes,1,opt,name=repository" json:"repository,omitempty"`
	Redirect   bool        `protobuf:"varint,2,opt,name=redirect" json:"redirect,omitempty"`
}

func (m *InitRepositoryRequest) Reset()         { *m = InitRepositoryRequest{} }
func (m *InitRepositoryRequest) String() string { return proto.CompactTextString(m) }
func (*InitRepositoryRequest) ProtoMessage()    {}

func (m *InitRepositoryRequest) GetRepository() *Repository {
	if m != nil {
		return m.Repository
	}
	return nil
}

type InitRepositoryResponse struct {
}

func (m *InitRepositoryResponse) Reset()         { *m = InitRepositoryResponse{} }
func (m *InitRepositoryResponse) String() string { return proto.CompactTextString(m) }
func (*InitRepositoryResponse) ProtoMessage()    {}

type GetFileRequest struct {
	Path *Path `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
}

func (m *GetFileRequest) Reset()         { *m = GetFileRequest{} }
func (m *GetFileRequest) String() string { return proto.CompactTextString(m) }
func (*GetFileRequest) ProtoMessage()    {}

func (m *GetFileRequest) GetPath() *Path {
	if m != nil {
		return m.Path
	}
	return nil
}

type MakeDirectoryRequest struct {
	Path     *Path `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	Redirect bool  `protobuf:"varint,2,opt,name=redirect" json:"redirect,omitempty"`
}

func (m *MakeDirectoryRequest) Reset()         { *m = MakeDirectoryRequest{} }
func (m *MakeDirectoryRequest) String() string { return proto.CompactTextString(m) }
func (*MakeDirectoryRequest) ProtoMessage()    {}

func (m *MakeDirectoryRequest) GetPath() *Path {
	if m != nil {
		return m.Path
	}
	return nil
}

type MakeDirectoryResponse struct {
}

func (m *MakeDirectoryResponse) Reset()         { *m = MakeDirectoryResponse{} }
func (m *MakeDirectoryResponse) String() string { return proto.CompactTextString(m) }
func (*MakeDirectoryResponse) ProtoMessage()    {}

type PutFileRequest struct {
	Path  *Path  `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *PutFileRequest) Reset()         { *m = PutFileRequest{} }
func (m *PutFileRequest) String() string { return proto.CompactTextString(m) }
func (*PutFileRequest) ProtoMessage()    {}

func (m *PutFileRequest) GetPath() *Path {
	if m != nil {
		return m.Path
	}
	return nil
}

type PutFileResponse struct {
}

func (m *PutFileResponse) Reset()         { *m = PutFileResponse{} }
func (m *PutFileResponse) String() string { return proto.CompactTextString(m) }
func (*PutFileResponse) ProtoMessage()    {}

type ListFilesRequest struct {
	Path  *Path  `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	Shard *Shard `protobuf:"bytes,2,opt,name=shard" json:"shard,omitempty"`
}

func (m *ListFilesRequest) Reset()         { *m = ListFilesRequest{} }
func (m *ListFilesRequest) String() string { return proto.CompactTextString(m) }
func (*ListFilesRequest) ProtoMessage()    {}

func (m *ListFilesRequest) GetPath() *Path {
	if m != nil {
		return m.Path
	}
	return nil
}

func (m *ListFilesRequest) GetShard() *Shard {
	if m != nil {
		return m.Shard
	}
	return nil
}

type ListFilesResponse struct {
	Path []*Path `protobuf:"bytes,1,rep,name=path" json:"path,omitempty"`
}

func (m *ListFilesResponse) Reset()         { *m = ListFilesResponse{} }
func (m *ListFilesResponse) String() string { return proto.CompactTextString(m) }
func (*ListFilesResponse) ProtoMessage()    {}

func (m *ListFilesResponse) GetPath() []*Path {
	if m != nil {
		return m.Path
	}
	return nil
}

type GetParentRequest struct {
	Commit *Commit `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
}

func (m *GetParentRequest) Reset()         { *m = GetParentRequest{} }
func (m *GetParentRequest) String() string { return proto.CompactTextString(m) }
func (*GetParentRequest) ProtoMessage()    {}

func (m *GetParentRequest) GetCommit() *Commit {
	if m != nil {
		return m.Commit
	}
	return nil
}

type GetParentResponse struct {
	Commit *Commit `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
}

func (m *GetParentResponse) Reset()         { *m = GetParentResponse{} }
func (m *GetParentResponse) String() string { return proto.CompactTextString(m) }
func (*GetParentResponse) ProtoMessage()    {}

func (m *GetParentResponse) GetCommit() *Commit {
	if m != nil {
		return m.Commit
	}
	return nil
}

type BranchRequest struct {
	Commit          *Commit         `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
	WriteCommitType WriteCommitType `protobuf:"varint,2,opt,name=write_commit_type,enum=pfs.WriteCommitType" json:"write_commit_type,omitempty"`
}

func (m *BranchRequest) Reset()         { *m = BranchRequest{} }
func (m *BranchRequest) String() string { return proto.CompactTextString(m) }
func (*BranchRequest) ProtoMessage()    {}

func (m *BranchRequest) GetCommit() *Commit {
	if m != nil {
		return m.Commit
	}
	return nil
}

type BranchResponse struct {
	Commit *Commit `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
}

func (m *BranchResponse) Reset()         { *m = BranchResponse{} }
func (m *BranchResponse) String() string { return proto.CompactTextString(m) }
func (*BranchResponse) ProtoMessage()    {}

func (m *BranchResponse) GetCommit() *Commit {
	if m != nil {
		return m.Commit
	}
	return nil
}

type CommitRequest struct {
	Commit *Commit `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
}

func (m *CommitRequest) Reset()         { *m = CommitRequest{} }
func (m *CommitRequest) String() string { return proto.CompactTextString(m) }
func (*CommitRequest) ProtoMessage()    {}

func (m *CommitRequest) GetCommit() *Commit {
	if m != nil {
		return m.Commit
	}
	return nil
}

type CommitResponse struct {
}

func (m *CommitResponse) Reset()         { *m = CommitResponse{} }
func (m *CommitResponse) String() string { return proto.CompactTextString(m) }
func (*CommitResponse) ProtoMessage()    {}

type PullDiffRequest struct {
	Commit *Commit `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
	Shard  *Shard  `protobuf:"bytes,2,opt,name=shard" json:"shard,omitempty"`
}

func (m *PullDiffRequest) Reset()         { *m = PullDiffRequest{} }
func (m *PullDiffRequest) String() string { return proto.CompactTextString(m) }
func (*PullDiffRequest) ProtoMessage()    {}

func (m *PullDiffRequest) GetCommit() *Commit {
	if m != nil {
		return m.Commit
	}
	return nil
}

func (m *PullDiffRequest) GetShard() *Shard {
	if m != nil {
		return m.Shard
	}
	return nil
}

type PushDiffRequest struct {
	Commit     *Commit    `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
	Shard      *Shard     `protobuf:"bytes,2,opt,name=shard" json:"shard,omitempty"`
	DriverType DriverType `protobuf:"varint,3,opt,name=driver_type,enum=pfs.DriverType" json:"driver_type,omitempty"`
	Value      []byte     `protobuf:"bytes,4,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *PushDiffRequest) Reset()         { *m = PushDiffRequest{} }
func (m *PushDiffRequest) String() string { return proto.CompactTextString(m) }
func (*PushDiffRequest) ProtoMessage()    {}

func (m *PushDiffRequest) GetCommit() *Commit {
	if m != nil {
		return m.Commit
	}
	return nil
}

func (m *PushDiffRequest) GetShard() *Shard {
	if m != nil {
		return m.Shard
	}
	return nil
}

type PushDiffResponse struct {
}

func (m *PushDiffResponse) Reset()         { *m = PushDiffResponse{} }
func (m *PushDiffResponse) String() string { return proto.CompactTextString(m) }
func (*PushDiffResponse) ProtoMessage()    {}

type GetCommitInfoRequest struct {
	Commit *Commit `protobuf:"bytes,1,opt,name=commit" json:"commit,omitempty"`
}

func (m *GetCommitInfoRequest) Reset()         { *m = GetCommitInfoRequest{} }
func (m *GetCommitInfoRequest) String() string { return proto.CompactTextString(m) }
func (*GetCommitInfoRequest) ProtoMessage()    {}

func (m *GetCommitInfoRequest) GetCommit() *Commit {
	if m != nil {
		return m.Commit
	}
	return nil
}

type GetCommitInfoResponse struct {
	CommitInfo *CommitInfo `protobuf:"bytes,1,opt,name=commit_info" json:"commit_info,omitempty"`
}

func (m *GetCommitInfoResponse) Reset()         { *m = GetCommitInfoResponse{} }
func (m *GetCommitInfoResponse) String() string { return proto.CompactTextString(m) }
func (*GetCommitInfoResponse) ProtoMessage()    {}

func (m *GetCommitInfoResponse) GetCommitInfo() *CommitInfo {
	if m != nil {
		return m.CommitInfo
	}
	return nil
}

func init() {
	proto.RegisterEnum("pfs.DriverType", DriverType_name, DriverType_value)
	proto.RegisterEnum("pfs.CommitType", CommitType_name, CommitType_value)
	proto.RegisterEnum("pfs.WriteCommitType", WriteCommitType_name, WriteCommitType_value)
}

// Client API for Api service

type ApiClient interface {
	// InitRepository creates a new repository.
	// An error is returned if the specified repository already exists.
	// An error is returned if the specified driver is not supported.
	InitRepository(ctx context.Context, in *InitRepositoryRequest, opts ...grpc.CallOption) (*InitRepositoryResponse, error)
	// GetFile returns a byte stream of the specified file.
	GetFile(ctx context.Context, in *GetFileRequest, opts ...grpc.CallOption) (Api_GetFileClient, error)
	// MakeDirectory makes a directory on the file system.
	MakeDirectory(ctx context.Context, in *MakeDirectoryRequest, opts ...grpc.CallOption) (*MakeDirectoryResponse, error)
	// PutFile writes the specified file to PFS.
	// An error is returned if the specified commit is not a write commit.
	// An error is returned is the specified commit was not opened for putting.
	PutFile(ctx context.Context, in *PutFileRequest, opts ...grpc.CallOption) (*PutFileResponse, error)
	// ListFiles lists the files within a directory.
	// An error is returned if the specified path is not a directory.
	ListFiles(ctx context.Context, in *ListFilesRequest, opts ...grpc.CallOption) (*ListFilesResponse, error)
	// GetParent gets the parent commit ID of the specified commit.
	GetParent(ctx context.Context, in *GetParentRequest, opts ...grpc.CallOption) (*GetParentResponse, error)
	// Branch creates a new write commit from a base commit.
	// An error is returned if the base commit is not a read commit.
	Branch(ctx context.Context, in *BranchRequest, opts ...grpc.CallOption) (*BranchResponse, error)
	// Commit turns the specified write commit into a read commit.
	// An error is returned if the specified commit is not a write commit.
	// An error is returned if there are outstanding shards to be pushed.
	Commit(ctx context.Context, in *CommitRequest, opts ...grpc.CallOption) (*CommitResponse, error)
	// PullDiff pulls a binary stream of the diff from the specified
	// commit to the commit's parent.
	PullDiff(ctx context.Context, in *PullDiffRequest, opts ...grpc.CallOption) (Api_PullDiffClient, error)
	// Push diff pushes a diff from the specified commit
	// to the commit's parent.
	// An error is returned if the specified commit is not a write commit.
	// An error is returned if the specified commit was not opened for pushing.
	// An error is returned if the specified driver does not match the repository's driver.
	PushDiff(ctx context.Context, in *PushDiffRequest, opts ...grpc.CallOption) (*PushDiffResponse, error)
	// GetCommitInfo returns the CommitInfo for a commit.
	GetCommitInfo(ctx context.Context, in *GetCommitInfoRequest, opts ...grpc.CallOption) (*GetCommitInfoResponse, error)
}

type apiClient struct {
	cc *grpc.ClientConn
}

func NewApiClient(cc *grpc.ClientConn) ApiClient {
	return &apiClient{cc}
}

func (c *apiClient) InitRepository(ctx context.Context, in *InitRepositoryRequest, opts ...grpc.CallOption) (*InitRepositoryResponse, error) {
	out := new(InitRepositoryResponse)
	err := grpc.Invoke(ctx, "/pfs.Api/InitRepository", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) GetFile(ctx context.Context, in *GetFileRequest, opts ...grpc.CallOption) (Api_GetFileClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_Api_serviceDesc.Streams[0], c.cc, "/pfs.Api/GetFile", opts...)
	if err != nil {
		return nil, err
	}
	x := &apiGetFileClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Api_GetFileClient interface {
	Recv() (*google_protobuf.BytesValue, error)
	grpc.ClientStream
}

type apiGetFileClient struct {
	grpc.ClientStream
}

func (x *apiGetFileClient) Recv() (*google_protobuf.BytesValue, error) {
	m := new(google_protobuf.BytesValue)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *apiClient) MakeDirectory(ctx context.Context, in *MakeDirectoryRequest, opts ...grpc.CallOption) (*MakeDirectoryResponse, error) {
	out := new(MakeDirectoryResponse)
	err := grpc.Invoke(ctx, "/pfs.Api/MakeDirectory", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) PutFile(ctx context.Context, in *PutFileRequest, opts ...grpc.CallOption) (*PutFileResponse, error) {
	out := new(PutFileResponse)
	err := grpc.Invoke(ctx, "/pfs.Api/PutFile", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) ListFiles(ctx context.Context, in *ListFilesRequest, opts ...grpc.CallOption) (*ListFilesResponse, error) {
	out := new(ListFilesResponse)
	err := grpc.Invoke(ctx, "/pfs.Api/ListFiles", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) GetParent(ctx context.Context, in *GetParentRequest, opts ...grpc.CallOption) (*GetParentResponse, error) {
	out := new(GetParentResponse)
	err := grpc.Invoke(ctx, "/pfs.Api/GetParent", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) Branch(ctx context.Context, in *BranchRequest, opts ...grpc.CallOption) (*BranchResponse, error) {
	out := new(BranchResponse)
	err := grpc.Invoke(ctx, "/pfs.Api/Branch", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) Commit(ctx context.Context, in *CommitRequest, opts ...grpc.CallOption) (*CommitResponse, error) {
	out := new(CommitResponse)
	err := grpc.Invoke(ctx, "/pfs.Api/Commit", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) PullDiff(ctx context.Context, in *PullDiffRequest, opts ...grpc.CallOption) (Api_PullDiffClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_Api_serviceDesc.Streams[1], c.cc, "/pfs.Api/PullDiff", opts...)
	if err != nil {
		return nil, err
	}
	x := &apiPullDiffClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Api_PullDiffClient interface {
	Recv() (*google_protobuf.BytesValue, error)
	grpc.ClientStream
}

type apiPullDiffClient struct {
	grpc.ClientStream
}

func (x *apiPullDiffClient) Recv() (*google_protobuf.BytesValue, error) {
	m := new(google_protobuf.BytesValue)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *apiClient) PushDiff(ctx context.Context, in *PushDiffRequest, opts ...grpc.CallOption) (*PushDiffResponse, error) {
	out := new(PushDiffResponse)
	err := grpc.Invoke(ctx, "/pfs.Api/PushDiff", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *apiClient) GetCommitInfo(ctx context.Context, in *GetCommitInfoRequest, opts ...grpc.CallOption) (*GetCommitInfoResponse, error) {
	out := new(GetCommitInfoResponse)
	err := grpc.Invoke(ctx, "/pfs.Api/GetCommitInfo", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Api service

type ApiServer interface {
	// InitRepository creates a new repository.
	// An error is returned if the specified repository already exists.
	// An error is returned if the specified driver is not supported.
	InitRepository(context.Context, *InitRepositoryRequest) (*InitRepositoryResponse, error)
	// GetFile returns a byte stream of the specified file.
	GetFile(*GetFileRequest, Api_GetFileServer) error
	// MakeDirectory makes a directory on the file system.
	MakeDirectory(context.Context, *MakeDirectoryRequest) (*MakeDirectoryResponse, error)
	// PutFile writes the specified file to PFS.
	// An error is returned if the specified commit is not a write commit.
	// An error is returned is the specified commit was not opened for putting.
	PutFile(context.Context, *PutFileRequest) (*PutFileResponse, error)
	// ListFiles lists the files within a directory.
	// An error is returned if the specified path is not a directory.
	ListFiles(context.Context, *ListFilesRequest) (*ListFilesResponse, error)
	// GetParent gets the parent commit ID of the specified commit.
	GetParent(context.Context, *GetParentRequest) (*GetParentResponse, error)
	// Branch creates a new write commit from a base commit.
	// An error is returned if the base commit is not a read commit.
	Branch(context.Context, *BranchRequest) (*BranchResponse, error)
	// Commit turns the specified write commit into a read commit.
	// An error is returned if the specified commit is not a write commit.
	// An error is returned if there are outstanding shards to be pushed.
	Commit(context.Context, *CommitRequest) (*CommitResponse, error)
	// PullDiff pulls a binary stream of the diff from the specified
	// commit to the commit's parent.
	PullDiff(*PullDiffRequest, Api_PullDiffServer) error
	// Push diff pushes a diff from the specified commit
	// to the commit's parent.
	// An error is returned if the specified commit is not a write commit.
	// An error is returned if the specified commit was not opened for pushing.
	// An error is returned if the specified driver does not match the repository's driver.
	PushDiff(context.Context, *PushDiffRequest) (*PushDiffResponse, error)
	// GetCommitInfo returns the CommitInfo for a commit.
	GetCommitInfo(context.Context, *GetCommitInfoRequest) (*GetCommitInfoResponse, error)
}

func RegisterApiServer(s *grpc.Server, srv ApiServer) {
	s.RegisterService(&_Api_serviceDesc, srv)
}

func _Api_InitRepository_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(InitRepositoryRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(ApiServer).InitRepository(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Api_GetFile_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetFileRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ApiServer).GetFile(m, &apiGetFileServer{stream})
}

type Api_GetFileServer interface {
	Send(*google_protobuf.BytesValue) error
	grpc.ServerStream
}

type apiGetFileServer struct {
	grpc.ServerStream
}

func (x *apiGetFileServer) Send(m *google_protobuf.BytesValue) error {
	return x.ServerStream.SendMsg(m)
}

func _Api_MakeDirectory_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(MakeDirectoryRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(ApiServer).MakeDirectory(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Api_PutFile_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(PutFileRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(ApiServer).PutFile(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Api_ListFiles_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(ListFilesRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(ApiServer).ListFiles(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Api_GetParent_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(GetParentRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(ApiServer).GetParent(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Api_Branch_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(BranchRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(ApiServer).Branch(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Api_Commit_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(CommitRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(ApiServer).Commit(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Api_PullDiff_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(PullDiffRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ApiServer).PullDiff(m, &apiPullDiffServer{stream})
}

type Api_PullDiffServer interface {
	Send(*google_protobuf.BytesValue) error
	grpc.ServerStream
}

type apiPullDiffServer struct {
	grpc.ServerStream
}

func (x *apiPullDiffServer) Send(m *google_protobuf.BytesValue) error {
	return x.ServerStream.SendMsg(m)
}

func _Api_PushDiff_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(PushDiffRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(ApiServer).PushDiff(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Api_GetCommitInfo_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(GetCommitInfoRequest)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(ApiServer).GetCommitInfo(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

var _Api_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pfs.Api",
	HandlerType: (*ApiServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "InitRepository",
			Handler:    _Api_InitRepository_Handler,
		},
		{
			MethodName: "MakeDirectory",
			Handler:    _Api_MakeDirectory_Handler,
		},
		{
			MethodName: "PutFile",
			Handler:    _Api_PutFile_Handler,
		},
		{
			MethodName: "ListFiles",
			Handler:    _Api_ListFiles_Handler,
		},
		{
			MethodName: "GetParent",
			Handler:    _Api_GetParent_Handler,
		},
		{
			MethodName: "Branch",
			Handler:    _Api_Branch_Handler,
		},
		{
			MethodName: "Commit",
			Handler:    _Api_Commit_Handler,
		},
		{
			MethodName: "PushDiff",
			Handler:    _Api_PushDiff_Handler,
		},
		{
			MethodName: "GetCommitInfo",
			Handler:    _Api_GetCommitInfo_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetFile",
			Handler:       _Api_GetFile_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "PullDiff",
			Handler:       _Api_PullDiff_Handler,
			ServerStreams: true,
		},
	},
}
