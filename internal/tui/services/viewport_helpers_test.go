package services

import "testing"

func TestNewStyledViewport_ClampsWidth(t *testing.T) {
	vp := NewStyledViewport(40, 10)
	if vp.Width() < 80 {
		t.Errorf("expected width >= 80, got %d", vp.Width())
	}
}

func TestNewStyledViewport_ClampsHeight(t *testing.T) {
	vp := NewStyledViewport(100, -5)
	if vp.Height() < 1 {
		t.Errorf("expected height >= 1, got %d", vp.Height())
	}
}

func TestNewStyledViewport_MouseWheel(t *testing.T) {
	vp := NewStyledViewport(100, 20)
	if !vp.MouseWheelEnabled {
		t.Error("expected MouseWheelEnabled to be true")
	}
}

func TestNewStyledViewport_SoftWrap(t *testing.T) {
	vp := NewStyledViewport(100, 20)
	if !vp.SoftWrap {
		t.Error("expected SoftWrap to be true")
	}
}

func TestNewStyledViewport_NormalValues(t *testing.T) {
	vp := NewStyledViewport(120, 30)
	if vp.Width() != 120 {
		t.Errorf("expected width 120, got %d", vp.Width())
	}
	if vp.Height() != 30 {
		t.Errorf("expected height 30, got %d", vp.Height())
	}
}
