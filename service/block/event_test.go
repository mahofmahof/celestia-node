package block

import (
	"context"
	"testing"
	"time"

	md "github.com/ipfs/go-merkledag/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/celestia-node/service/header"
	core "github.com/tendermint/tendermint/types"
)

// TestEventLoop tests that the Service event loop spawned by calling
// `Start` on the Service properly listens for new blocks from its Fetcher
// and handles them accordingly.
func TestEventLoop(t *testing.T) {
	mockFetcher := &mockFetcher{
		mockNewBlockCh: make(chan *RawBlock),
	}
	serv := NewBlockService(mockFetcher, md.Mock())

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	err := serv.Start(ctx)
	require.NoError(t, err)

	numBlocks := 3
	expectedBlocks := mockFetcher.generateBlocks(t, numBlocks)

	// as we calculate extended square twice, this may take a lot of time, causing the main thread expect the result
	// earlier than its ready.
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < numBlocks; i++ {
		block, err := serv.GetBlockData(ctx, expectedBlocks[i].Header().DAH)
		require.NoError(t, err)
		assert.Equal(t, expectedBlocks[i].data.Width(), block.Width())
		assert.Equal(t, expectedBlocks[i].data.RowRoots(), block.RowRoots())
		assert.Equal(t, expectedBlocks[i].data.ColRoots(), block.ColRoots())
	}

	err = serv.Stop(ctx)
	require.NoError(t, err)
}

// mockFetcher mocks away the `Fetcher` interface.
type mockFetcher struct {
	mockNewBlockCh chan *RawBlock
}

func (m *mockFetcher) GetBlock(ctx context.Context, height *int64) (*RawBlock, error) {
	return nil, nil
}

func (m *mockFetcher) Commit(ctx context.Context, height *int64) (*core.Commit, error) {
	return nil, nil
}

func (m *mockFetcher) ValidatorSet(ctx context.Context, height *int64) (*core.ValidatorSet, error) {
	return nil, nil
}

func (m *mockFetcher) SubscribeNewBlockEvent(ctx context.Context) (<-chan *RawBlock, error) {
	return m.mockNewBlockCh, nil
}

func (m *mockFetcher) UnsubscribeNewBlockEvent(ctx context.Context) error {
	close(m.mockNewBlockCh)
	return nil
}

// generateBlocks generates new raw blocks and sends them to the mock fetcher,
// returning the extended blocks generated from the process to compare against.
func (m *mockFetcher) generateBlocks(t *testing.T, num int) []Block {
	t.Helper()

	extendedBlocks := make([]Block, num)

	for i := 0; i < num; i++ {
		rawBlock, block := generateRawAndExtendedBlock(t)
		extendedBlocks[i] = *block
		m.mockNewBlockCh <- rawBlock
	}

	return extendedBlocks
}

func generateRawAndExtendedBlock(t *testing.T) (*RawBlock, *Block) {
	t.Helper()

	data, err := GenerateRandomBlockData(1, 1, 1, 1, 40)
	if err != nil {
		t.Fatal(err)
	}
	rawBlock := &RawBlock{
		Data: data,
	}
	// extend the data to get the data hash
	extendedData, err := extendBlockData(rawBlock)
	if err != nil {
		t.Fatal(err)
	}
	dah, err := header.DataAvailabilityHeaderFromExtendedData(extendedData)
	if err != nil {
		t.Fatal(err)
	}
	rawBlock.Header = header.RawHeader{
		DataHash: dah.Hash(),
	}
	return rawBlock, &Block{
		header: &header.ExtendedHeader{
			DAH: &dah,
		},
		data: extendedData,
	}
}
