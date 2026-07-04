package flex_pay

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestComputeSettlementHash pins the H5 idempotency key: the hash must be
// stable across re-uploads of the same request set and order-independent
// (sorted before hashing), while distinct request sets must hash distinctly.
func TestComputeSettlementHash_StableAndOrderIndependent(t *testing.T) {
	t.Parallel()

	shuffled, err := computeSettlementHash([]uint64{3, 1, 2})
	require.NoError(t, err)
	sorted, err := computeSettlementHash([]uint64{1, 2, 3})
	require.NoError(t, err)
	require.Equal(t, sorted, shuffled, "hash must be order-independent (input is sorted)")

	dupOrder, err := computeSettlementHash([]uint64{2, 3, 1})
	require.NoError(t, err)
	require.Equal(t, sorted, dupOrder, "any permutation of the same set must hash equally")

	other, err := computeSettlementHash([]uint64{1, 2, 4})
	require.NoError(t, err)
	require.NotEqual(t, sorted, other, "different request sets must hash differently")

	empty, err := computeSettlementHash([]uint64{})
	require.NoError(t, err)
	nonEmpty, err := computeSettlementHash([]uint64{1})
	require.NoError(t, err)
	require.NotEqual(t, empty, nonEmpty, "empty vs non-empty must differ")
}
