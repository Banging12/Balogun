// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// from bcValidBlockTest.json, "SimpleTx"
func TestBlockEncoding(t *testing.T) {
	blockEnc := common.FromHex("f90262f9023ca07285abd5b24742f184ad676e31f6054663b3529bc35ea2fcad8a3e0f642a46f7a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347948888f1f195afa192cfee860698584c030f4c9db1a0ecc60e00b3fe5ce9f6e1a10e5469764daf51f1fe93c22ec3f9a7583a80357217a0b771728ba50b7514869ad38dab045d60063540a492764460fcd902f9f73cf4a2a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421b90100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008302000001832fefd8825208845c47775c80a00000000000000000000000000000000000000000000000000000000000000000880000000000000000b8413e835d97c294e9c5a24e072eb6cb07c91f562042c4dac5e67229b5cea30b35c73258e1a426173c4267c480a13d7c8c0c92ca70d55837b706194b763e0d57186a01e1e0800a82c3508094095e7baea6a6c7c4c2dfeb977efac326af552d870a80808080c0")
	var block Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		t.Fatal("decode error: ", err)
	}

	check := func(f string, got, want interface{}) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mismatch: got %v, want %v", f, got, want)
		}
	}
	check("Difficulty", block.Difficulty(), big.NewInt(131072))
	check("GasLimit", block.GasLimit(), uint64(3141592))
	check("GasUsed", block.GasUsed(), uint64(21000))
	check("Coinbase", block.Coinbase(), common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"))
	check("MixDigest", block.MixDigest(), common.HexToHash("0000000000000000000000000000000000000000000000000000000000000000"))
	check("Root", block.Root(), common.HexToHash("ecc60e00b3fe5ce9f6e1a10e5469764daf51f1fe93c22ec3f9a7583a80357217"))
	check("Hash", block.Hash(), common.HexToHash("683656aec2ca2a1ff1f0378577c15ea903e862fc39002c4355059c9b0d2cc610"))
	check("Nonce", block.Nonce(), uint64(0x0))
	check("Time", block.Time(), big.NewInt(1548187484))
	check("Size", block.Size(), common.StorageSize(len(blockEnc)))
	check("ParentHash", block.ParentHash(), common.HexToHash("7285abd5b24742f184ad676e31f6054663b3529bc35ea2fcad8a3e0f642a46f7"))
	check("Signature", block.Signature(), common.FromHex("3e835d97c294e9c5a24e072eb6cb07c91f562042c4dac5e67229b5cea30b35c73258e1a426173c4267c480a13d7c8c0c92ca70d55837b706194b763e0d57186a01"))

	check("len(Transactions)", len(block.Transactions()), 1)
	check("Transactions[0].Hash", block.Transactions()[0].Hash(), common.HexToHash("e51163e3f878dcf71a17f7b006798fef634ba1f214883d87224ce6582e26d3a7"))

	ourBlockEnc, err := rlp.EncodeToBytes(&block)
	if err != nil {
		t.Fatal("encode error: ", err)
	}
	if !bytes.Equal(ourBlockEnc, blockEnc) {
		t.Errorf("encoded block mismatch:\ngot:  %x\nwant: %x", ourBlockEnc, blockEnc)
	}
}
