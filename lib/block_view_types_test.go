package lib

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/deso-protocol/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Initialize empty DeSoEncoders and check if they are encoded properly.
func TestEmptyTypeEncoders(t *testing.T) {
	require := require.New(t)
	testCases := _getAllDeSoEncoders(t)

	// And now try to encode/decode for the empty encoders.
	for _, testType := range testCases {
		if testType.GetEncoderType() == EncoderTypeBlock {
			testType = &MsgDeSoBlock{Header: &MsgDeSoHeader{}}
		} else if testType.GetEncoderType() == EncoderTypeTxn {
			testType = &MsgDeSoTxn{
				TxnMeta: &BasicTransferMetadata{},
			}
		} else if testType.GetEncoderType() == EncoderTypeBlockNode {
			testType = &BlockNode{
				Hash:             &BlockHash{},
				DifficultyTarget: &BlockHash{},
				CumWork:          big.NewInt(0),
				Header:           &MsgDeSoHeader{},
			}
		}
		testBytes := EncodeToBytes(0, testType)
		rr := bytes.NewReader(testBytes)
		exists, err := DecodeFromBytes(testType, rr)
		require.Equal(true, exists)
		require.NoError(err)
	}
}

// Randomly initialize DeSoEncoders using gofakeit package and check if they are encoded properly.
func TestRandomTypeEncoders(t *testing.T) {
	require := require.New(t)
	_ = require

	// Make sure encoder migrations are not triggered yet.
	for ii := range GlobalDeSoParams.EncoderMigrationHeightsList {
		if GlobalDeSoParams.EncoderMigrationHeightsList[ii].Version == 0 {
			continue
		}
		GlobalDeSoParams.EncoderMigrationHeightsList[ii].Height = 1
	}

	encodeCases := _getAllDeSoEncoders(t)
	decodeCases := _getAllDeSoEncoders(t)
	// Make sure the encoder migration for v3 messages is tested.
	GlobalDeSoParams.ForkHeights = RegtestForkHeights
	for ii := range encodeCases {
		// State change entry encoder is tested separately in TestStateChangeEntryEncoder.
		if encodeCases[ii].GetEncoderType() == EncoderTypeStateChangeEntry {
			continue
		}
		gofakeit.Struct(encodeCases[ii])
		// Only certain block and transaction configurations are valid, so they are hardcoded here.
		// Block and transaction serialization is tested extensively in network_test.go.
		if encodeCases[ii].GetEncoderType() == EncoderTypeBlock {
			encodeCases[ii] = &MsgDeSoBlock{Header: &MsgDeSoHeader{}}
			decodeCases[ii] = &MsgDeSoBlock{Header: &MsgDeSoHeader{}}
		} else if encodeCases[ii].GetEncoderType() == EncoderTypeTxn {
			encodeCases[ii] = &MsgDeSoTxn{
				TxnMeta: &BasicTransferMetadata{},
			}
			decodeCases[ii] = &MsgDeSoTxn{
				TxnMeta: &BasicTransferMetadata{},
			}
		} else if encodeCases[ii].GetEncoderType() == EncoderTypeBlockNode {
			encodeCases[ii] = &BlockNode{
				Hash:             &BlockHash{},
				DifficultyTarget: &BlockHash{},
				CumWork:          big.NewInt(0),
				Header:           &MsgDeSoHeader{},
			}
			decodeCases[ii] = &BlockNode{
				Hash:             &BlockHash{},
				DifficultyTarget: &BlockHash{},
				CumWork:          big.NewInt(0),
				Header:           &MsgDeSoHeader{},
			}
		}
		encodedBytes := EncodeToBytes(0, encodeCases[ii])
		rr := bytes.NewReader(encodedBytes)
		exists, err := DecodeFromBytes(decodeCases[ii], rr)
		if exists != true {
			t.Fatalf("Encode and decode exists is false! Entry type: %v, err: %v",
				encodeCases[ii].GetEncoderType(), err)
		}
		require.NoError(err)
		reEncodedBytes := EncodeToBytes(0, decodeCases[ii])
		if reflect.DeepEqual(encodedBytes, reEncodedBytes) != true {
			t.Fatalf("Encode and decode doesn't match! Entry type: %v", encodeCases[ii].GetEncoderType())
		}
	}
}

// Get an array of all DeSo encoders.
func _getAllDeSoEncoders(t *testing.T) []DeSoEncoder {
	var encoders []DeSoEncoder

	// First add all block view DeSoEncoders
	initialTypeBlockView := EncoderTypeUtxoEntry
	for initialTypeBlockView != EncoderTypeEndBlockView {
		encoders = append(encoders, initialTypeBlockView.New())
		initialTypeBlockView += 1
	}

	// Now add all txindex DeSoEncoders
	initialTypeTxIndex := EncoderTypeTransactionMetadata
	for initialTypeTxIndex != EncoderTypeEndTxIndex {
		encoders = append(encoders, initialTypeTxIndex.New())
		initialTypeTxIndex += 1
	}
	return encoders
}

func TestMessageEntryDecoding(t *testing.T) {
	// Create a message entry
	messageEntry := &MessageEntry{
		NewPublicKey(m0PkBytes),
		NewPublicKey(m1PkBytes),
		[]byte{1, 2, 3, 4, 5, 6},
		uint64(time.Now().UnixNano()),
		false,
		MessagesVersion1,
		NewPublicKey(m0PkBytes),
		NewGroupKeyName([]byte("default")),
		NewPublicKey(m1PkBytes),
		BaseGroupKeyName(),
		nil,
	}

	encodedWithExtraData := EncodeToBytes(0, messageEntry)

	// We know the last byte is a 0 representing the length of the extra data, so chop that off
	missingExtraDataEncoding := encodedWithExtraData[:len(encodedWithExtraData)-1]

	decodedMessageEntryMissingExtraData := &MessageEntry{}
	rr := bytes.NewReader(missingExtraDataEncoding)
	exists, err := DecodeFromBytes(decodedMessageEntryMissingExtraData, rr)
	require.Equal(t, true, exists)
	require.NoError(t, err)

	decodedMessageEntryWithExtraData := &MessageEntry{}
	rr = bytes.NewReader(encodedWithExtraData)
	exists, err = DecodeFromBytes(decodedMessageEntryWithExtraData, rr)
	require.Equal(t, true, exists)
	require.NoError(t, err)

	// The message decoded without extra data should
	require.True(t, reflect.DeepEqual(decodedMessageEntryWithExtraData, decodedMessageEntryMissingExtraData))
	require.True(t, reflect.DeepEqual(decodedMessageEntryMissingExtraData, messageEntry))

	// Now encode them again and prove they're the same
	require.True(t, bytes.Equal(encodedWithExtraData, EncodeToBytes(0, decodedMessageEntryMissingExtraData)))

	// Okay now let's set the extra data on the message entry
	messageEntry.ExtraData = map[string][]byte{
		"test": {0, 1, 2},
	}

	encodedExtraData := EncodeExtraData(messageEntry.ExtraData)

	encodedIncludingExtraData := EncodeToBytes(0, messageEntry)

	extraDataBytesRemoved := encodedIncludingExtraData[:len(encodedIncludingExtraData)-len(encodedExtraData)]

	messageEntryWithExtraDataRemoved := &MessageEntry{}
	rr = bytes.NewReader(extraDataBytesRemoved)
	exists, err = DecodeFromBytes(messageEntryWithExtraDataRemoved, rr)
	require.Equal(t, true, exists)
	require.NoError(t, err)

	messageEntryWithExtraDataRemovedBytes := EncodeToBytes(0, messageEntryWithExtraDataRemoved)

	// This should be effectively equivalent to the original message entry above without extra data
	require.True(t, reflect.DeepEqual(messageEntryWithExtraDataRemoved, decodedMessageEntryWithExtraData))

	// The bytes should be the same up until the extra data segment of the bytes
	require.Equal(t, len(encodedIncludingExtraData), len(messageEntryWithExtraDataRemovedBytes)+len(encodedExtraData)-1)
	reflect.DeepEqual(encodedIncludingExtraData, append(messageEntryWithExtraDataRemovedBytes[:len(messageEntryWithExtraDataRemovedBytes)-1], encodedExtraData...))
}

func TestMessagingGroupEntryDecoding(t *testing.T) {
	// Create a messaging group entry

	messagingGroupEntry := &MessagingGroupEntry{
		GroupOwnerPublicKey:   NewPublicKey(m0PkBytes),
		MessagingPublicKey:    NewPublicKey(m0PkBytes),
		MessagingGroupKeyName: BaseGroupKeyName(),
	}

	encodedWithExtraData := EncodeToBytes(0, messagingGroupEntry)

	// We know the last byte is a 0 representing the length of the extra data, so chop that off
	missingExtraDataEncoding := encodedWithExtraData[:len(encodedWithExtraData)-1]

	decodedMessagingGroupEntryMissingExtraData := &MessagingGroupEntry{}
	rr := bytes.NewReader(missingExtraDataEncoding)
	exists, err := DecodeFromBytes(decodedMessagingGroupEntryMissingExtraData, rr)
	require.Equal(t, true, exists)
	require.NoError(t, err)

	decodedMessagingGroupEntryWithExtraData := &MessagingGroupEntry{}
	rr = bytes.NewReader(encodedWithExtraData)
	exists, err = DecodeFromBytes(decodedMessagingGroupEntryWithExtraData, rr)
	require.Equal(t, true, exists)
	require.NoError(t, err)

	// The message decoded without extra data should
	require.True(t, reflect.DeepEqual(decodedMessagingGroupEntryWithExtraData, decodedMessagingGroupEntryMissingExtraData))
	require.True(t, reflect.DeepEqual(decodedMessagingGroupEntryMissingExtraData, messagingGroupEntry))

	// Now encode them again and prove they're the same
	require.True(t, bytes.Equal(encodedWithExtraData, EncodeToBytes(0, decodedMessagingGroupEntryMissingExtraData)))

	// Okay now let's set the extra data on the message entry
	messagingGroupEntry.ExtraData = map[string][]byte{
		"test": {0, 1, 2},
	}

	encodedExtraData := EncodeExtraData(messagingGroupEntry.ExtraData)

	encodedIncludingExtraData := EncodeToBytes(0, messagingGroupEntry)

	extraDataBytesRemoved := encodedIncludingExtraData[:len(encodedIncludingExtraData)-len(encodedExtraData)]

	messagingGroupEntryWithExtraDataRemoved := &MessagingGroupEntry{}
	rr = bytes.NewReader(extraDataBytesRemoved)
	exists, err = DecodeFromBytes(messagingGroupEntryWithExtraDataRemoved, rr)
	require.Equal(t, true, exists)
	require.NoError(t, err)

	messagingGroupEntryWithExtraDataRemovedBytes := EncodeToBytes(0, messagingGroupEntryWithExtraDataRemoved)

	// This should be effectively equivalent to the original message entry above without extra data
	require.True(t, reflect.DeepEqual(messagingGroupEntryWithExtraDataRemoved, decodedMessagingGroupEntryWithExtraData))

	// The bytes should be the same up until the extra data segment of the bytes
	require.Equal(t, len(encodedIncludingExtraData), len(messagingGroupEntryWithExtraDataRemovedBytes)+len(encodedExtraData)-1)
	reflect.DeepEqual(encodedIncludingExtraData, append(messagingGroupEntryWithExtraDataRemovedBytes[:len(messagingGroupEntryWithExtraDataRemovedBytes)-1], encodedExtraData...))
}

// A lazy test based on TestBitcoinExchange to check utxo encoding/decoding.
func TestUtxoEntryEncodeDecode(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	_ = assert
	_ = require

	// Don't refresh the universal view for this test, since it causes a race condition
	// to trigger.
	// TODO: Lower this value to .1 and fix this race condition.
	ReadOnlyUtxoViewRegenerationIntervalSeconds = 100

	oldInitialUSDCentsPerBitcoinExchangeRate := InitialUSDCentsPerBitcoinExchangeRate
	InitialUSDCentsPerBitcoinExchangeRate = uint64(1350000)
	defer func() {
		InitialUSDCentsPerBitcoinExchangeRate = oldInitialUSDCentsPerBitcoinExchangeRate
	}()

	paramsTmp := DeSoTestnetParams
	paramsTmp.DeSoNanosPurchasedAtGenesis = 0
	chain, params, db := NewLowDifficultyBlockchainWithParams(t, &paramsTmp)
	mempool, _ := NewTestMiner(t, chain, params, true /*isSender*/)

	// Read in the test Bitcoin blocks and headers.
	bitcoinBlocks, bitcoinHeaders, bitcoinHeaderHeights := _readBitcoinExchangeTestData(t)

	// Extract BitcoinExchange transactions from the test Bitcoin blocks.
	bitcoinExchangeTxns := []*MsgDeSoTxn{}
	for _, block := range bitcoinBlocks {
		currentBurnTxns, err :=
			ExtractBitcoinExchangeTransactionsFromBitcoinBlock(
				block, BitcoinTestnetBurnAddress, params)
		require.NoError(err)
		bitcoinExchangeTxns = append(bitcoinExchangeTxns, currentBurnTxns...)
	}

	// Verify that Bitcoin burn transactions are properly extracted from Bitcoin blocks
	// and the their burn amounts are computed correctly.
	require.Equal(9, len(bitcoinExchangeTxns))

	minBurnBlocks := uint32(2)
	startHeaderIndex := 0
	paramsCopy := GetTestParamsCopy(
		bitcoinHeaders[startHeaderIndex], bitcoinHeaderHeights[startHeaderIndex],
		params, minBurnBlocks,
	)
	paramsCopy.BitcoinBurnAddress = BitcoinTestnetBurnAddress
	chain.params = paramsCopy
	// Reset the pool to give the mempool access to the new BitcoinManager object.
	newMP := NewDeSoMempool(chain, 0, /* rateLimitFeeRateNanosPerKB */
		0, /* minFeeRateNanosPerKB */
		"" /*blockCypherAPIKey*/, false,
		"" /*dataDir*/, "", true)
	mempool.resetPool(newMP)

	// Validating the first Bitcoin burn transaction via a UtxoView should
	// fail because the block corresponding to it is not yet in the BitcoinManager.
	burnTxn1 := bitcoinExchangeTxns[0]
	burnTxn2 := bitcoinExchangeTxns[1]

	// Applying the full transaction with its merkle proof should work.
	{
		mempoolTxs, err := mempool.processTransaction(
			burnTxn1, true /*allowUnconnectedTxn*/, true /*rateLimit*/, 0, /*peerID*/
			true /*verifySignatures*/)
		require.NoError(err)
		require.Equal(1, len(mempoolTxs))
		require.Equal(1, len(mempool.poolMap))
		mempoolTxRet := mempool.poolMap[*mempoolTxs[0].Hash]
		require.Equal(
			mempoolTxRet.Tx.TxnMeta.(*BitcoinExchangeMetadata),
			burnTxn1.TxnMeta.(*BitcoinExchangeMetadata))
	}

	// According to the mempool, the balance of the user whose public key created
	// the Bitcoin burn transaction should now have some DeSo.
	pkBytes1, _ := hex.DecodeString(BitcoinTestnetPub1)
	pkBytes3, _ := hex.DecodeString(BitcoinTestnetPub3)

	// The mempool should be able to process a burn transaction directly.
	{
		mempoolTxsAdded, err := mempool.processTransaction(
			burnTxn2, true /*allowUnconnectedTxn*/, true /*rateLimit*/, 0, /*peerID*/
			true /*verifySignatures*/)
		require.NoError(err)
		require.Equal(1, len(mempoolTxsAdded))
		require.Equal(2, len(mempool.poolMap))
		mempoolTxRet := mempool.poolMap[*mempoolTxsAdded[0].Hash]
		require.Equal(
			mempoolTxRet.Tx.TxnMeta.(*BitcoinExchangeMetadata),
			burnTxn2.TxnMeta.(*BitcoinExchangeMetadata))
	}

	// According to the mempool, the balances should have updated.
	{
		utxoEntries, err := chain.GetSpendableUtxosForPublicKey(pkBytes3, mempool, nil)
		require.NoError(err)

		require.Equal(1, len(utxoEntries))
		assert.Equal(int64(3372255472), int64(utxoEntries[0].AmountNanos))
	}

	// If the mempool is not consulted, the balances should be zero.
	{
		utxoEntries, err := chain.GetSpendableUtxosForPublicKey(pkBytes3, nil, nil)
		require.NoError(err)
		require.Equal(0, len(utxoEntries))
	}

	// Make the moneyPkString the paramUpdater so they can update the exchange rate.
	rateUpdateIndex := 4
	params.ExtraRegtestParamUpdaterKeys = make(map[PkMapKey]bool)
	params.ExtraRegtestParamUpdaterKeys[MakePkMapKey(MustBase58CheckDecode(moneyPkString))] = true
	paramsCopy.ExtraRegtestParamUpdaterKeys = params.ExtraRegtestParamUpdaterKeys
	// Applying all the txns to the UtxoView should work. Include a rate update
	// in the middle.
	utxoOpsList := [][]*UtxoOperation{}
	{
		utxoView := NewUtxoView(db, paramsCopy, nil, chain.snapshot, nil)

		// Add a placeholder where the rate update is going to be
		fff := append([]*MsgDeSoTxn{}, bitcoinExchangeTxns[:rateUpdateIndex]...)
		fff = append(fff, nil)
		fff = append(fff, bitcoinExchangeTxns[rateUpdateIndex:]...)
		bitcoinExchangeTxns = fff

		for ii := range bitcoinExchangeTxns {
			// When we hit the rate update, populate the placeholder.
			if ii == rateUpdateIndex {
				newUSDCentsPerBitcoin := uint64(27000 * 100)
				_, rateUpdateTxn, _, err := _updateUSDCentsPerBitcoinExchangeRate(
					t, chain, db, params, 100, /*feeRateNanosPerKB*/
					moneyPkString,
					moneyPrivString,
					newUSDCentsPerBitcoin)
				require.NoError(err)

				bitcoinExchangeTxns[ii] = rateUpdateTxn
				burnTxn := bitcoinExchangeTxns[ii]
				blockHeight := chain.blockTip().Height + 1
				utxoOps, totalInput, totalOutput, fees, err :=
					utxoView.ConnectTransaction(burnTxn, burnTxn.Hash(), blockHeight, 0, true, false)
				_, _, _ = totalInput, totalOutput, fees
				require.NoError(err)
				utxoOpsList = append(utxoOpsList, utxoOps)
				continue
			}

			burnTxn := bitcoinExchangeTxns[ii]
			blockHeight := chain.blockTip().Height + 1
			utxoOps, totalInput, totalOutput, fees, err :=
				utxoView.ConnectTransaction(burnTxn, burnTxn.Hash(), blockHeight, 0, true, false)
			require.NoError(err)

			require.Equal(2, len(utxoOps))
			//fmt.Println(int64(totalInput), ",")
			assert.Equal(int64(fees), int64(totalInput-totalOutput))

			_, _, _ = ii, totalOutput, fees
			utxoOpsList = append(utxoOpsList, utxoOps)
		}
		utxoEntries, err := chain.GetSpendableUtxosForPublicKey(pkBytes1, nil, utxoView)
		require.NoError(err)
		for _, entry := range utxoEntries {
			entryBytes := EncodeToBytes(0, entry)
			newEntry := &UtxoEntry{}
			rr := bytes.NewReader(entryBytes)
			DecodeFromBytes(newEntry, rr)
			require.Equal(reflect.DeepEqual(entry.String(), newEntry.String()), true)
		}

		// Flushing the UtxoView should work.
		require.NoError(utxoView.FlushToDb(0))
	}
	t.Cleanup(func() {
		if !newMP.stopped {
			newMP.Stop()
		}
	})
}

func TestEncodingUint256s(t *testing.T) {
	// Create three uint256.Ints.
	num1 := uint256.NewInt(0)
	num2 := uint256.NewInt(598128756)
	num3 := MaxUint256

	// Encode them to bytes using VariableEncodeUint256.
	encoded1 := VariableEncodeUint256(num1)
	encoded2 := VariableEncodeUint256(num2)
	encoded3 := VariableEncodeUint256(num3)

	// Decode them from bytes using VariableDecodeUint256. Verify values.
	rr := bytes.NewReader(encoded1)
	decoded1, err := VariableDecodeUint256(rr)
	require.NoError(t, err)
	require.True(t, num1.Eq(decoded1))

	rr = bytes.NewReader(encoded2)
	decoded2, err := VariableDecodeUint256(rr)
	require.NoError(t, err)
	require.True(t, num2.Eq(decoded2))

	rr = bytes.NewReader(encoded3)
	decoded3, err := VariableDecodeUint256(rr)
	require.NoError(t, err)
	require.True(t, num3.Eq(decoded3))

	// Test that VariableEncodeUint256 does not provide a fixed-width byte encoding.
	require.NotEqual(t, len(encoded1), len(encoded2))
	require.NotEqual(t, len(encoded1), len(encoded3))

	// Encode them to bytes using FixedWidthEncodeUint256.
	encoded1 = FixedWidthEncodeUint256(num1)
	encoded2 = FixedWidthEncodeUint256(num2)
	encoded3 = FixedWidthEncodeUint256(num3)

	// Decode them from bytes using FixedWidthDecodeUint256. Verify values.
	rr = bytes.NewReader(encoded1)
	decoded1, err = FixedWidthDecodeUint256(rr)
	require.NoError(t, err)
	require.True(t, num1.Eq(decoded1))

	rr = bytes.NewReader(encoded2)
	decoded2, err = FixedWidthDecodeUint256(rr)
	require.NoError(t, err)
	require.True(t, num2.Eq(decoded2))

	rr = bytes.NewReader(encoded3)
	decoded3, err = FixedWidthDecodeUint256(rr)
	require.NoError(t, err)
	require.True(t, num3.Eq(decoded3))

	// Test that FixedWidthEncodeUint256 provides a fixed-width byte encoding.
	require.Equal(t, len(encoded1), len(encoded2))
	require.Equal(t, len(encoded1), len(encoded3))
}
