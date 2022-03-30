/*
	Package gitbom implements GitBOM.

	Read the spec at https://hackmd.io/@aeva/draft-gitbom-spec

	GitBOM is neither git nor an SBOM.


	It is an application of the git DAG, a widely used merkle tree with a flat-file storage format, to the challenge of creating build artifact trees in today’s language-heterogeneous open source environments.
	by generating artifact trees at build time, embedding the hash of the tree in produced artifacts, and referencing that hash in the next build step, GitBOM will enable the zero-end-user-effort creation of verifiable build trees. Furthermore, it will enable launch-time comparison of vulnerability data against a complete artifact tree for both open source and proprietary projects (if vuln data is traceable back to source file).

	Objective
	It is desirable to enable efficient launch-time comparison of the verifiable and complete build tree of any executable component [1] against a then-current list of undesirable source files [2] which are known to be undesirable, where such a build tree contains unique referents for all sources from which the given executable object was composed.

	[1]: binary, dynamically-linked library, container image, etc.

	[2]: because vulnerabilities may be discovered between the time an executable is created and the time when it is run, these processes must be decoupled
*/
package gitbom

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"sort"
	"strings"
	"sync"
)

// ArtifactTree provides a common interface that assists with the creation and management of a GitBOM document.
type ArtifactTree interface {
	Identifier

	// AddSha1Reference adds a SHA1 based git reference to the current GitBOM document.
	// obj []byte is the byte array to be tagged in the GitRef.
	// bom Identifier is the gitbom identifier of the artifact tree used to create the object.
	// The resulting reference is based on the GitRef format.
	// It returns an error if the SHA1 implementation fails.
	AddSha1Reference(obj []byte, bom Identifier) error

	// AddSha1ReferenceFromReader adds a SHA1 based git reference to the current GitBOM document.
	// The resulting reference is based on the GitRef format.
	// The io.Reader will be continuously be read until the reader returns a non-null error.
	// If the io.Reader returns io.EOF, the read is considered to be complete.
	// Any other return value from Reader is an error.
	// The object length must be included.
	// If the amount of bytes read does not match the stated object length, an error is returned.
	AddSha1ReferenceFromReader(reader io.Reader, bom Identifier, objLength int64) error

	// AddSha256Reference see AddSha1Reference. A SHA256 hasher is used instead of SHA1.
	AddSha256Reference(obj []byte, bom Identifier) error

	// AddSha256ReferenceFromReader see AddSha1ReferenceFromReader. A SHA256 hasher is used instead of SHA1.
	AddSha256ReferenceFromReader(reader io.Reader, bom Identifier, objLength int64) error

	// References Returns a lsit of references in the order it will be printed.
	References() []Reference

	// Sha1GitRef returns the SHA1 GitRef of this object as a hex string.
	Sha1GitRef() string

	// Sha256GitRef returns the SHA256 GitRef of this object as a hex string.
	Sha256GitRef() string

	// String Returns the string representation of the GitBOM.
	String() string
}

type Reference interface {
	// Identity returns the GitRef identity of the object as a hex string.
	Identity() string

	// Bom returns an Identifier representing the dependency tree of the object represented by the Identity
	Bom() Identifier

	// String returns a ArtifactTree entry represented by this Reference.
	String() string
}

func referenceSorter(r1, r2 Reference) bool {
	return r1.Identity() < r2.Identity()
}

type by func(p1, p2 Reference) bool

func (b by) sort(refs []Reference) {
	sorter := &referenceSort{
		refs: refs,
		by:   b,
	}
	sort.Sort(sorter)
}

type reference struct {
	identity string
	bom      Identifier
}

type referenceSort struct {
	refs []Reference
	by   by
}

func (grs *referenceSort) Len() int {
	return len(grs.refs)
}

func (grs *referenceSort) Swap(i, j int) {
	grs.refs[i], grs.refs[j] = grs.refs[j], grs.refs[i]
}

func (grs *referenceSort) Less(i, j int) bool {
	return grs.by(grs.refs[i], grs.refs[j])
}

func (ref reference) Identity() string {
	return ref.identity
}

func (ref reference) Bom() Identifier {
	return ref.Bom()
}

func (ref reference) String() string {
	res := fmt.Sprintf("blob %s", ref.identity)
	if ref.bom != nil {
		res = fmt.Sprintf("%s bom %s", res, ref.bom.Identity())
	}

	res = res + "\n"
	return res
}

type Identifier interface {
	Identity() string
}

type gitBom struct {
	lock    sync.Mutex
	gitRefs []Reference
}

// NewGitBom creates a new ArtifactTree object.
// Thread Safety: none, apply your own controls.
//
// Adding duplicate objects with the same Reference identity results in only one Reference entry.
// References are sorted in ascending order based on their UTF-8 values.
//
// Implementation details:
// Adding a Reference is O(n) to discover duplicates.
// Generating a ArtifactTree is O(n*log(n)) as it sorts the existing refs.
func NewGitBom() ArtifactTree {
	return &gitBom{}
}

func (srv *gitBom) AddSha1Reference(obj []byte, bom Identifier) error {
	hashAlgorithm := sha1.New()
	reader := bytes.NewBuffer(obj)
	return srv.addGitRef(reader, bom, hashAlgorithm, int64(len(obj)))
}

func (srv *gitBom) AddSha256Reference(obj []byte, bom Identifier) error {
	hashAlgorithm := sha256.New()
	reader := bytes.NewBuffer(obj)
	return srv.addGitRef(reader, bom, hashAlgorithm, int64(len(obj)))
}

func (srv *gitBom) AddSha1ReferenceFromReader(reader io.Reader, bom Identifier, objLength int64) error {
	hashAlgorithm := sha1.New()
	return srv.addGitRef(reader, bom, hashAlgorithm, objLength)
}

func (srv *gitBom) AddSha256ReferenceFromReader(reader io.Reader, bom Identifier, objLength int64) error {
	hashAlgorithm := sha256.New()
	return srv.addGitRef(reader, bom, hashAlgorithm, objLength)
}

func (srv *gitBom) addGitRef(reader io.Reader, bom Identifier, hashAlgorithm hash.Hash, length int64) error {
	identity, err := generateGitHash(reader, hashAlgorithm, length)
	if err != nil {
		return err
	}

	ref := reference{
		identity: identity,
		bom:      bom,
	}

	srv.lock.Lock()
	srv.gitRefs = append(srv.gitRefs, ref)
	srv.lock.Unlock()
	return nil
}

func (srv *gitBom) References() []Reference {
	srv.lock.Lock()
	by(referenceSorter).sort(srv.gitRefs)
	result := make([]Reference, 0, len(srv.gitRefs))
	for _, ref := range srv.gitRefs {
		result = append(result, ref)
	}
	srv.lock.Unlock()
	return srv.gitRefs
}

func (srv *gitBom) String() string {
	srv.lock.Lock()
	by(referenceSorter).sort(srv.gitRefs)
	refs := make([]string, 0)
	for _, ref := range srv.gitRefs {
		refs = append(refs, ref.String())
	}
	srv.lock.Unlock()
	return strings.Join(refs, "")
}

func (srv *gitBom) Sha1GitRef() string {
	generated := srv.String()
	hashAlgorithm := sha1.New()
	res, err := generateGitHash(bytes.NewBuffer([]byte(generated)), hashAlgorithm, int64(len(generated)))
	if err != nil {
		// we should only see this if the runtime was fundamentally broken
		panic(err)
	}
	return res
}

func (srv *gitBom) Sha256GitRef() string {
	generated := srv.String()
	hashAlgorithm := sha256.New()
	res, err := generateGitHash(bytes.NewBuffer([]byte(generated)), hashAlgorithm, int64(len(generated)))
	if err != nil {
		// we should only see this if the runtime was fundamentally broken
		panic(err)
	}
	return res
}

func (srv *gitBom) Identity() string {
	return srv.Sha256GitRef()
}

func generateGitHash(reader io.Reader, hashAlgorithm hash.Hash, length int64) (string, error) {
	// \u0000 is the unicode sequence for '\0'
	header := fmt.Sprintf("blob %d\u0000", length)
	n, err := hashAlgorithm.Write([]byte(header))
	if err != nil {
		return "", err
	}
	if n != len(header) {
		return "", errors.New("impartial header write while generating git ref")
	}

	written, err := io.Copy(hashAlgorithm, reader)
	if err != nil {
		return "", err
	}
	if written != length {
		return "", errors.New("actual reader length did not match expected length")
	}

	hashBytes := hashAlgorithm.Sum([]byte{})
	hashStr := hex.EncodeToString(hashBytes)
	return hashStr, nil
}

type identifier struct {
	identity string
}

func (gb identifier) Identity() string {
	return gb.identity
}

func NewIdentifier(identity string) (Identifier, error) {
	// TODO check if gitbom matches the format
	_, err := hex.DecodeString(identity)
	if err != nil {
		return nil, err
	}
	return &identifier{
		identity: identity,
	}, nil
}
