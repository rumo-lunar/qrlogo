package qr_test

import (
	"testing"

	"github.com/rumo-lunar/qrlogo/qr"
)

func TestEncodeRS_KnownHandTraced(t *testing.T) {
	// Arrange: data = [1, 1, 1, 1], numEC = 2.
	//
	// g_2(x) = x² + 3x + 2.
	// Hand-traced polynomial long division gives EC = [20, 20].
	data := []byte{1, 1, 1, 1}

	// Act
	ec := qr.EncodeRS(data, 2)

	// Assert
	if len(ec) != 2 {
		t.Fatalf("len(ec) = %d, want 2", len(ec))
	}
	want := []byte{20, 20}
	for i, w := range want {
		if ec[i] != w {
			t.Errorf("EC[%d] = %d, want %d", i, ec[i], w)
		}
	}
}

func TestEncodeRS_AllZeroIsAllZero(t *testing.T) {
	// Arrange: zero input → zero EC, regardless of numEC.
	ec := qr.EncodeRS(make([]byte, 30), 28)

	for i, b := range ec {
		if b != 0 {
			t.Errorf("EC[%d] = %d, want 0", i, b)
		}
	}
}

func TestEncodeRS_PanicsOnZeroNumEC(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on numEC=0")
		}
	}()
	qr.EncodeRS([]byte{1}, 0)
}

func TestEncodeRS_PanicsOnEmptyData(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on empty data")
		}
	}()
	qr.EncodeRS(nil, 10)
}
