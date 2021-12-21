package gitbom

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSha1GitRef(t *testing.T) {
	buf := bytes.NewBufferString("hello world")

	hash, err := generateGitHash(buf, sha1.New(), 11)
	assert.NoError(t, err)
	assert.Equal(t, "95d09f2b10159347eece71399a7e2e907ea3df4f", hash)
}

func TestNewSha256GitRef(t *testing.T) {
	buf := bytes.NewBufferString("hello world")

	hash, err := generateGitHash(buf, sha256.New(), 11)
	assert.NoError(t, err)
	assert.Equal(t, "fee53a18d32820613c0527aa79be5cb30173c823a9b448fa4817767cc84c6f03", hash)
}

func TestGitRef_ShortRead(t *testing.T) {
	buf := bytes.NewBufferString("hello world")

	hash, err := generateGitHash(buf, sha1.New(), 12)
	assert.Error(t, err)
	assert.Equal(t, "", hash)
}

func TestGitRef_LongRead(t *testing.T) {
	buf := bytes.NewBufferString("hello world")

	hash, err := generateGitHash(buf, sha1.New(), 10)
	assert.Error(t, err)
	assert.Equal(t, "", hash)
}

func TestFlatWorkflow(t *testing.T) {
	string1 := "hello"
	string2 := "world"

	gb := NewGitBom()
	err := gb.AddSha1ReferenceFromReader(bytes.NewBufferString(string1), nil, int64(len(string1)))
	assert.NoError(t, err)
	err = gb.AddSha1ReferenceFromReader(bytes.NewBufferString(string2), nil, int64(len(string2)))
	assert.NoError(t, err)
	expected := "blob 04fea06420ca60892f73becee3614f6d023a4b7f\n" +
		"blob b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0\n"
	assert.Equal(t, expected, gb.String())

	ref := gb.Sha1GitRef()
	assert.NoError(t, err)
	assert.Equal(t, "dc0be356e8c2ba26e66448d97db76ad050206574", ref)

}
func TestNestedWorkflow(t *testing.T) {
	string1 := "hello"
	string2 := "world"

	gb := NewGitBom()
	err := gb.AddSha1ReferenceFromReader(bytes.NewBufferString(string1), nil, int64(len(string1)))
	assert.NoError(t, err)
	err = gb.AddSha1ReferenceFromReader(bytes.NewBufferString(string2), nil, int64(len(string2)))
	assert.NoError(t, err)
	expected := "blob 04fea06420ca60892f73becee3614f6d023a4b7f\n" +
		"blob b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0\n"
	assert.Equal(t, expected, gb.String())

	ref := gb.Sha1GitRef()
	assert.Equal(t, "dc0be356e8c2ba26e66448d97db76ad050206574", ref)

	string3 := "hello2"
	string4 := "independent"
	string5 := "opaque"

	gb2 := NewGitBom()

	err = gb2.AddSha1Reference([]byte(string3), gb)
	assert.NoError(t, err)

	err = gb2.AddSha1Reference([]byte(string4), nil)
	assert.NoError(t, err)
	assert.Equal(t, "blob 23294b0610492cf55c1c4835216f20d376a287dd bom 588ed637c6073a58e79f4fc63a85158eafed022a2b791f7765c28a3c3d1797d6\nblob be78cc5602c5457f144a67e574b8f98b9dc2a1a0\n", gb2.String())

	opaqueBom, err := NewOpaqueGitBom("a87d2b20b13568a5530ec6a59dacfdda8ee3cd1e3d63c9d13da26d27e3447812")
	assert.NoError(t, err)
	err = gb2.AddSha256Reference([]byte(string5), opaqueBom)
	assert.NoError(t, err)
	assert.Equal(t, "blob 23294b0610492cf55c1c4835216f20d376a287dd bom 588ed637c6073a58e79f4fc63a85158eafed022a2b791f7765c28a3c3d1797d6\nblob be78cc5602c5457f144a67e574b8f98b9dc2a1a0\nblob dcf17826ff7a346e6b09704314fb5ef4c9fcceb85c2936b45cab13bc7167991a bom a87d2b20b13568a5530ec6a59dacfdda8ee3cd1e3d63c9d13da26d27e3447812\n", gb2.String())
}

func TestInvalidOpaqueGitBom(t *testing.T) {
	_, err := NewOpaqueGitBom("23294b0610492cf55c1c4835216f20d376a287dd")
	assert.NoError(t, err)
}

func TestInvalidOpaqueGitBom_TooFewCharacters(t *testing.T) {
	_, err := NewOpaqueGitBom("23294b0610492cf55c1c4835216f20d376a287d")
	assert.Error(t, err)
}

func TestInvalidOpaqueGitBom_NonHexCharacter(t *testing.T) {
	_, err := NewOpaqueGitBom("23294b0610492cf55c1c4835216f20d376a287dg")
	assert.Error(t, err)
}

func TestInvalidOpaqueGitBom_ExtraTrailingSpace(t *testing.T) {
	_, err := NewOpaqueGitBom("23294b0610492cf55c1c4835216f20d376a287dd ")
	assert.Error(t, err)
}

func TestInvalidOpaqueGitBom_ExtraSpaces(t *testing.T) {
	_, err := NewOpaqueGitBom(" 23294b0610492cf55c1c4835216f20d376a287dd ")
	assert.Error(t, err)
}

func TestInvalidOpaqueGitBom_VeryInvalid(t *testing.T) {
	_, err := NewOpaqueGitBom(" 23294b0610492cf 55c1c4835216f20d376a287dd ")
	assert.Error(t, err)
}
