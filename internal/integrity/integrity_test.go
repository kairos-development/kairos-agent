package integrity

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashExcludingAttestationBlock_WithBlock(t *testing.T) {
	base := []byte("abcKAIROS_ATTEST_BEGINpayloadKAIROS_ATTEST_ENDxyz")
	expected := HashExcludingAttestationBlock([]byte("abcxyz"))
	actual := HashExcludingAttestationBlock(base)

	assert.Equal(t, expected, actual)
}

func TestHashExcludingAttestationBlock_WithoutBlock(t *testing.T) {
	data := []byte("simple data without attestation block")
	hash := HashExcludingAttestationBlock(data)

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA256 hex = 64 chars
}

func TestHashExcludingAttestationBlock_EmptyData(t *testing.T) {
	hash := HashExcludingAttestationBlock([]byte{})
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)
}

func TestHashExcludingAttestationBlock_OnlyBeginMarker(t *testing.T) {
	data := []byte("dataKAIROS_ATTEST_BEGINmore data")
	hash := HashExcludingAttestationBlock(data)

	// Should not exclude anything if end marker is missing
	assert.NotEmpty(t, hash)
}

func TestHashExcludingAttestationBlock_OnlyEndMarker(t *testing.T) {
	data := []byte("dataKAIROS_ATTEST_ENDmore data")
	hash := HashExcludingAttestationBlock(data)

	// Should not exclude anything if begin marker is missing
	assert.NotEmpty(t, hash)
}

func TestHashExcludingAttestationBlock_MultipleBlocks(t *testing.T) {
	// Only first block should be excluded
	data := []byte("KAIROS_ATTEST_BEGINblock1KAIROS_ATTEST_ENDKAIROS_ATTEST_BEGINblock2KAIROS_ATTEST_END")
	hash := HashExcludingAttestationBlock(data)

	assert.NotEmpty(t, hash)
}

func TestHashExcludingAttestationBlock_EmptyBlock(t *testing.T) {
	data := []byte("beforeKAIROS_ATTEST_BEGINKAIROS_ATTEST_ENDafter")
	expected := HashExcludingAttestationBlock([]byte("beforeafter"))
	actual := HashExcludingAttestationBlock(data)

	assert.Equal(t, expected, actual)
}

func TestHashExcludingAttestationBlock_BlockAtStart(t *testing.T) {
	data := []byte("KAIROS_ATTEST_BEGINpayloadKAIROS_ATTEST_ENDafter")
	expected := HashExcludingAttestationBlock([]byte("after"))
	actual := HashExcludingAttestationBlock(data)

	assert.Equal(t, expected, actual)
}

func TestHashExcludingAttestationBlock_BlockAtEnd(t *testing.T) {
	data := []byte("beforeKAIROS_ATTEST_BEGINpayloadKAIROS_ATTEST_END")
	expected := HashExcludingAttestationBlock([]byte("before"))
	actual := HashExcludingAttestationBlock(data)

	assert.Equal(t, expected, actual)
}

func TestHashExcludingAttestationBlock_Deterministic(t *testing.T) {
	data := []byte("test data")
	hash1 := HashExcludingAttestationBlock(data)
	hash2 := HashExcludingAttestationBlock(data)

	assert.Equal(t, hash1, hash2)
}

func TestHashExcludingAttestationBlock_DifferentData(t *testing.T) {
	hash1 := HashExcludingAttestationBlock([]byte("data1"))
	hash2 := HashExcludingAttestationBlock([]byte("data2"))

	assert.NotEqual(t, hash1, hash2)
}

func TestVerifyBinary_Trusted(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_binary_*.bin")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	data := []byte("test binary data")
	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	tmpFile.Close()

	expectedHash := HashExcludingAttestationBlock(data)

	status, err := VerifyBinary(tmpFile.Name(), expectedHash)
	require.NoError(t, err)
	assert.Equal(t, StatusTrusted, status)
}

func TestVerifyBinary_Degraded(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_binary_*.bin")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	data := []byte("test binary data")
	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	tmpFile.Close()

	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	status, err := VerifyBinary(tmpFile.Name(), wrongHash)
	require.NoError(t, err)
	assert.Equal(t, StatusDegraded, status)
}

func TestVerifyBinary_FileNotFound(t *testing.T) {
	status, err := VerifyBinary("/nonexistent/binary", "somehash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read binary")
	assert.Equal(t, StatusDegraded, status)
}

func TestVerifyBinary_WithAttestationBlock(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_binary_*.bin")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	data := []byte("beforeKAIROS_ATTEST_BEGINattestationKAIROS_ATTEST_ENDafter")
	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	tmpFile.Close()

	// Hash should match data without attestation block
	expectedHash := HashExcludingAttestationBlock([]byte("beforeafter"))

	status, err := VerifyBinary(tmpFile.Name(), expectedHash)
	require.NoError(t, err)
	assert.Equal(t, StatusTrusted, status)
}

func TestVerifyBinary_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_binary_*.bin")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	expectedHash := HashExcludingAttestationBlock([]byte{})

	status, err := VerifyBinary(tmpFile.Name(), expectedHash)
	require.NoError(t, err)
	assert.Equal(t, StatusTrusted, status)
}

func TestVerifyBinary_LargeFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_binary_*.bin")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Create 1MB file
	data := make([]byte, 1<<20)
	for i := range data {
		data[i] = byte(i % 256)
	}
	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	tmpFile.Close()

	expectedHash := HashExcludingAttestationBlock(data)

	status, err := VerifyBinary(tmpFile.Name(), expectedHash)
	require.NoError(t, err)
	assert.Equal(t, StatusTrusted, status)
}

func TestIndexOf_Found(t *testing.T) {
	body := []byte("hello world")
	needle := []byte("world")

	index := indexOf(body, needle)
	assert.Equal(t, 6, index)
}

func TestIndexOf_NotFound(t *testing.T) {
	body := []byte("hello world")
	needle := []byte("xyz")

	index := indexOf(body, needle)
	assert.Equal(t, -1, index)
}

func TestIndexOf_AtStart(t *testing.T) {
	body := []byte("hello world")
	needle := []byte("hello")

	index := indexOf(body, needle)
	assert.Equal(t, 0, index)
}

func TestIndexOf_AtEnd(t *testing.T) {
	body := []byte("hello world")
	needle := []byte("world")

	index := indexOf(body, needle)
	assert.Equal(t, 6, index)
}

func TestIndexOf_EmptyNeedle(t *testing.T) {
	body := []byte("hello world")
	needle := []byte{}

	index := indexOf(body, needle)
	assert.Equal(t, 0, index)
}

func TestIndexOf_EmptyBody(t *testing.T) {
	body := []byte{}
	needle := []byte("hello")

	index := indexOf(body, needle)
	assert.Equal(t, -1, index)
}

func TestIndexOf_NeedleLongerThanBody(t *testing.T) {
	body := []byte("hi")
	needle := []byte("hello")

	index := indexOf(body, needle)
	assert.Equal(t, -1, index)
}

func TestIndexOf_MultipleOccurrences(t *testing.T) {
	body := []byte("hello hello world")
	needle := []byte("hello")

	// Should return first occurrence
	index := indexOf(body, needle)
	assert.Equal(t, 0, index)
}

func TestIndexOf_PartialMatch(t *testing.T) {
	body := []byte("helhello world")
	needle := []byte("hello")

	index := indexOf(body, needle)
	assert.Equal(t, 3, index)
}

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, Status("trusted"), StatusTrusted)
	assert.Equal(t, Status("degraded"), StatusDegraded)
}

func TestVerifyBinary_InvalidPath(t *testing.T) {
	status, err := VerifyBinary("", "somehash")
	assert.Error(t, err)
	assert.Equal(t, StatusDegraded, status)
}

func TestVerifyBinary_DirectoryPath(t *testing.T) {
	tmpDir := t.TempDir()
	status, err := VerifyBinary(tmpDir, "somehash")
	assert.Error(t, err)
	assert.Equal(t, StatusDegraded, status)
}

func TestHashExcludingAttestationBlock_LargeData(t *testing.T) {
	// Create 1MB data with attestation block in the middle
	before := make([]byte, 500000)
	after := make([]byte, 500000)
	for i := range before {
		before[i] = byte(i % 256)
	}
	for i := range after {
		after[i] = byte((i + 100) % 256)
	}

	data := append(before, []byte("KAIROS_ATTEST_BEGINattestationKAIROS_ATTEST_END")...)
	data = append(data, after...)

	expectedData := append(before, after...)
	expected := HashExcludingAttestationBlock(expectedData)
	actual := HashExcludingAttestationBlock(data)

	assert.Equal(t, expected, actual)
}
