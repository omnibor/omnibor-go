package gitbom

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSha1GitRef(t *testing.T) {
	buf := bytes.NewBufferString("hello world")

	hasher := sha1.New()

	hash, err := generateGitHash(buf, 11, hasher)
	assert.NoError(t, err)
	assert.Equal(t, "95d09f2b10159347eece71399a7e2e907ea3df4f", hash)
}

func TestNewSha256GitRef(t *testing.T) {
	buf := bytes.NewBufferString("hello world")

	hash, err := generateGitHash(buf, 11, sha256.New())
	assert.NoError(t, err)
	assert.Equal(t, "fee53a18d32820613c0527aa79be5cb30173c823a9b448fa4817767cc84c6f03", hash)
}

func TestNewSha1AndSha256GitRef(t *testing.T) {
	buf := bytes.NewBufferString("hello world")

	hash, err := generateGitHash(buf, 11, sha1.New(), sha256.New())
	assert.NoError(t, err)
	assert.Equal(t, "95d09f2b10159347eece71399a7e2e907ea3df4f+fee53a18d32820613c0527aa79be5cb30173c823a9b448fa4817767cc84c6f03", hash)
}

func TestGitRef_ShortRead(t *testing.T) {
	buf := bytes.NewBufferString("hello world")

	hash, err := generateGitHash(buf, 12, sha1.New())
	assert.Error(t, err)
	assert.Equal(t, "", hash)
}

func TestGitRef_LongRead(t *testing.T) {
	buf := bytes.NewBufferString("hello world")

	hash, err := generateGitHash(buf, 10, sha1.New())
	assert.Error(t, err)
	assert.Equal(t, "", hash)
}

// TODO create sha256 version
func TestFlatWorkflow(t *testing.T) {
	string1 := "hello"
	string2 := "world"

	gb := NewSha1GitBom()
	err := gb.AddReferenceFromReader(bytes.NewBufferString(string1), nil, int64(len(string1)))
	assert.NoError(t, err)
	err = gb.AddReferenceFromReader(bytes.NewBufferString(string2), nil, int64(len(string2)))
	assert.NoError(t, err)
	expected := "blob 04fea06420ca60892f73becee3614f6d023a4b7f\n" +
		"blob b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0\n"
	assert.Equal(t, expected, gb.String())

	ref := gb.Identity()
	assert.NoError(t, err)

	assert.Equal(t, "dc0be356e8c2ba26e66448d97db76ad050206574", ref)
}

// TODO add sha256
func TestNestedWorkflow(t *testing.T) {
	string1 := "hello"
	string2 := "world"

	gb := NewSha1GitBom()
	err := gb.AddReferenceFromReader(bytes.NewBufferString(string1), nil, int64(len(string1)))
	assert.NoError(t, err)
	err = gb.AddReferenceFromReader(bytes.NewBufferString(string2), nil, int64(len(string2)))
	assert.NoError(t, err)
	expected := "blob 04fea06420ca60892f73becee3614f6d023a4b7f\nblob b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0\n"

	assert.Equal(t, expected, gb.String())

	ref := gb.Identity()
	expected = "dc0be356e8c2ba26e66448d97db76ad050206574"

	assert.Equal(t, expected, ref)

	string3 := "hello2"
	string4 := "independent"
	string5 := "opaque"

	gb2 := NewSha1GitBom()

	err = gb2.AddReference([]byte(string3), gb)
	assert.NoError(t, err)

	err = gb2.AddReference([]byte(string4), nil)
	assert.NoError(t, err)
	expected = "blob 23294b0610492cf55c1c4835216f20d376a287dd bom dc0be356e8c2ba26e66448d97db76ad050206574\nblob be78cc5602c5457f144a67e574b8f98b9dc2a1a0\n"

	assert.Equal(t, expected, gb2.String())

	identifier, err := NewIdentifier("a87d2b20b13568a5530ec6a59dacfdda8ee3cd1e3d63c9d13da26d27e3447812")
	assert.NoError(t, err)
	err = gb2.AddReference([]byte(string5), identifier)
	assert.NoError(t, err)
	expected = "blob 23294b0610492cf55c1c4835216f20d376a287dd bom dc0be356e8c2ba26e66448d97db76ad050206574\nblob 32898208a218272b0fa7549f60951d4eed2ed830 bom a87d2b20b13568a5530ec6a59dacfdda8ee3cd1e3d63c9d13da26d27e3447812\nblob be78cc5602c5457f144a67e574b8f98b9dc2a1a0\n"

	assert.Equal(t, expected, gb2.String())
}

func TestValidIdentifier(t *testing.T) {
	_, err := NewIdentifier("23294b0610492cf55c1c4835216f20d376a287dd")
	assert.NoError(t, err)
}

func TestInvalidIdentifier_TooFewCharacters(t *testing.T) {
	_, err := NewIdentifier("23294b0610492cf55c1c4835216f20d376a287d")
	assert.Error(t, err)
}

func TestInvalidIdentifier_NonHexCharacter(t *testing.T) {
	_, err := NewIdentifier("23294b0610492cf55c1c4835216f20d376a287dg")
	assert.Error(t, err)
}

func TestInvalidIdentifier_ExtraTrailingSpace(t *testing.T) {
	_, err := NewIdentifier("23294b0610492cf55c1c4835216f20d376a287dd ")
	assert.Error(t, err)
}

func TestInvalidIdentifier_ExtraSpaces(t *testing.T) {
	_, err := NewIdentifier(" 23294b0610492cf55c1c4835216f20d376a287dd ")
	assert.Error(t, err)
}

func TestInvalidIdentifier_VeryInvalid(t *testing.T) {
	_, err := NewIdentifier(" 23294b0610492cf 55c1c4835216f20d376a287dd ")
	assert.Error(t, err)
}

func BenchmarkNewGitBom(b *testing.B) {
	dataset := generateDataset(b.N)

	gb := NewSha1GitBom()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//fmt.Println(dataset[i])
		_ = gb.AddReference(dataset[i], nil)
	}
	b.StopTimer()

	fmt.Println(len(gb.References()), len(dataset), b.N)
}

func generateDataset(n int) [][]byte {
	dataset := make([][]byte, 0)
	for i := 0; i < n; i++ {
		buf := make([]byte, 64)
		binary.LittleEndian.PutUint32(buf, uint32(i))
		//fmt.Println(buf)
		dataset = append(dataset, buf)
	}
	for i := 0; i < len(dataset); i++ {
		//fmt.Println(dataset[i])
	}
	return dataset
}
