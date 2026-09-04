package services

import "testing"

func TestLabelRateTarget(t *testing.T) {
	flatRates := map[string]int{
		"phổ thông.ngày thường.NT":    66666,
		"phổ thông.ngày nghỉ.NT":      100000,
		"kinh nghiệm.ngày thường.NT":  70000,
		"phổ thông.ngày nghỉ.T7":      100000,
		"phổ thông.ngày nghỉ.OT":      0, // zero-rate buckets must not resolve
		"phổ thông.ngày thường.OT T7": 100000,
	}
	got, ok := labelRateTarget(flatRates, "nt")
	if !ok || got.dayType != "ngày thường" || got.hourType != "NT" {
		t.Fatalf("labelRateTarget(nt) = %+v ok=%v, want ngày thường/NT", got, ok)
	}
	got, ok = labelRateTarget(flatRates, "T7")
	if !ok || got.dayType != "ngày nghỉ" || got.hourType != "T7" {
		t.Fatalf("labelRateTarget(T7) = %+v ok=%v, want ngày nghỉ/T7", got, ok)
	}
	got, ok = labelRateTarget(flatRates, "OT T7")
	if !ok || got.hourType != "OT T7" {
		t.Fatalf("labelRateTarget(OT T7) = %+v ok=%v", got, ok)
	}
	if _, ok := labelRateTarget(flatRates, "ZZ"); ok {
		t.Fatal("unknown label must not resolve")
	}
	if _, ok := labelRateTarget(flatRates, "OT"); ok {
		t.Fatal("zero-rate bucket must not resolve")
	}
}
