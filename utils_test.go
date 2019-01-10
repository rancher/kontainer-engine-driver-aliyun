package main

import "testing"

func TestVersionGreaterThan(t *testing.T) {
	VersionMap := []struct {
		New string
		Old string
	}{
		{
			New: "1.11.5",
			Old: "1.9.10",
		},
		{
			New: "1.11.5",
			Old: "1.10.11",
		},
		{
			New: "1.12.4-aliyun.1",
			Old: "1.11.5",
		},
		{
			New: "1.12.4-aliyun.1",
			Old: "1.11.5",
		},
		{
			New: "1.12.4-aliyun.1",
			Old: "1.12.3",
		},
		{
			New: "1.12",
			Old: "1.11.5",
		},
		{
			New: "1.12.1",
			Old: "",
		},
	}

	for _, v := range VersionMap {
		if versionGreaterThan(v.Old, v.New) {
			t.Fatalf("unexpected %s>%s", v.Old, v.New)
		}
		if !versionGreaterThan(v.New, v.Old) {
			t.Fatalf("unexpected %s<=%s", v.New, v.Old)
		}
	}
}
