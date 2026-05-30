package solana

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"

	"github.com/mr-tron/base58"
)

type PublicKey [32]byte

func MustPublicKeyFromBase58(s string) PublicKey {
	pk, err := PublicKeyFromBase58(s)
	if err != nil {
		panic(err)
	}
	return pk
}

func PublicKeyFromBase58(s string) (PublicKey, error) {
	decoded, err := base58.Decode(s)
	if err != nil {
		return PublicKey{}, err
	}
	if len(decoded) != 32 {
		return PublicKey{}, fmt.Errorf("invalid public key length: %d", len(decoded))
	}
	var pk PublicKey
	copy(pk[:], decoded)
	return pk, nil
}

func (pk PublicKey) String() string {
	return base58.Encode(pk[:])
}

func (pk PublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(pk.String())
}

func (pk *PublicKey) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	v, err := PublicKeyFromBase58(s)
	if err != nil {
		return err
	}
	*pk = v
	return nil
}

type PrivateKey []byte

func PrivateKeyFromBase58(s string) (PrivateKey, error) {
	return base58.Decode(s)
}

type Wallet struct {
	privateKey PrivateKey
	publicKey  PublicKey
}

func WalletFromBytes(privateKeyBytes []byte) (*Wallet, error) {
	if len(privateKeyBytes) != 64 {
		return nil, fmt.Errorf("invalid private key length: %d (expected 64)", len(privateKeyBytes))
	}
	priv := ed25519.PrivateKey(privateKeyBytes)
	pub := priv.Public().(ed25519.PublicKey)
	w := &Wallet{privateKey: privateKeyBytes}
	copy(w.publicKey[:], pub)
	return w, nil
}

func WalletFromPrivateKeyBase58(s string) (*Wallet, error) {
	decoded, err := base58.Decode(s)
	if err != nil {
		return nil, err
	}
	return WalletFromBytes(decoded)
}

func (w *Wallet) PublicKey() PublicKey {
	return w.publicKey
}

func (w *Wallet) PrivateKey() PrivateKey {
	return w.privateKey
}

func (w *Wallet) Sign(msg []byte) []byte {
	return ed25519.Sign(ed25519.PrivateKey(w.privateKey), msg)
}

func DLMMProgramID() PublicKey {
	return MustPublicKeyFromBase58("LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo")
}

func WSOL() PublicKey {
	return MustPublicKeyFromBase58("So11111111111111111111111111111111111111112")
}

const LAMPORTS_PER_SOL = 1_000_000_000

func LamportsToSol(lamports uint64) float64 {
	return float64(lamports) / LAMPORTS_PER_SOL
}

func SolToLamports(sol float64) uint64 {
	return uint64(sol * LAMPORTS_PER_SOL)
}
