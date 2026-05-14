package mfd

import (
	"encoding/xml"
	"testing"
)

func TestAttribute_EnumXMLRoundTrip(t *testing.T) {
	attr := Attribute{
		Name:       "Status",
		DBName:     "status",
		DBType:     "varchar",
		GoType:     "ScanStatus",
		IsEnum:     true,
		EnumType:   "scan_status",
		EnumValues: "pending,running,finished",
	}

	data, err := xml.MarshalIndent(attr, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var got Attribute
	if err := xml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	if !got.IsEnum {
		t.Error("IsEnum lost after round-trip")
	}
	if got.EnumType != "scan_status" {
		t.Errorf("EnumType = %q, want %q", got.EnumType, "scan_status")
	}
	if got.EnumValues != "pending,running,finished" {
		t.Errorf("EnumValues = %q, want %q", got.EnumValues, "pending,running,finished")
	}
}

func TestAttribute_EnumValuesList(t *testing.T) {
	attr := Attribute{EnumValues: "pending,running,finished"}
	vals := attr.EnumValuesList()
	if len(vals) != 3 || vals[0] != "pending" || vals[1] != "running" || vals[2] != "finished" {
		t.Errorf("EnumValuesList() = %v", vals)
	}

	empty := Attribute{}
	if v := empty.EnumValuesList(); v != nil {
		t.Errorf("expected nil, got %v", v)
	}
}

func TestAttribute_MergeEnum(t *testing.T) {
	t.Run("enum transition updates GoType", func(t *testing.T) {
		existing := &Attribute{
			Name:   "Status",
			DBName: "status",
			DBType: "varchar",
			GoType: "string",
		}

		from := &Attribute{
			Name:       "Status",
			DBName:     "status",
			DBType:     "varchar",
			GoType:     "ScanStatus",
			IsEnum:     true,
			EnumType:   "scan_status",
			EnumValues: "pending,running",
		}

		existing.Merge(from, false)

		if !existing.IsEnum {
			t.Error("IsEnum not propagated")
		}
		if existing.EnumType != "scan_status" {
			t.Errorf("EnumType = %q", existing.EnumType)
		}
		if existing.EnumValues != "pending,running" {
			t.Errorf("EnumValues = %q", existing.EnumValues)
		}
		if existing.GoType != "ScanStatus" {
			t.Errorf("GoType = %q, want ScanStatus", existing.GoType)
		}
	})

	t.Run("non-enum preserves GoType", func(t *testing.T) {
		existing := &Attribute{
			Name:   "Title",
			DBName: "title",
			DBType: "varchar",
			GoType: "string",
		}

		from := &Attribute{
			Name:   "Title",
			DBName: "title",
			DBType: "varchar",
			GoType: "string",
		}

		existing.Merge(from, false)

		if existing.GoType != "string" {
			t.Errorf("GoType = %q, want string", existing.GoType)
		}
	})

	t.Run("enum removal resets GoType", func(t *testing.T) {
		existing := &Attribute{
			Name:       "Status",
			DBName:     "status",
			DBType:     "varchar",
			GoType:     "ScanStatus",
			IsEnum:     true,
			EnumType:   "scan_status",
			EnumValues: "pending,running",
		}

		from := &Attribute{
			Name:   "Status",
			DBName: "status",
			DBType: "varchar",
			GoType: "string",
		}

		existing.Merge(from, false)

		if existing.IsEnum {
			t.Error("IsEnum should be false")
		}
		if existing.GoType != "string" {
			t.Errorf("GoType = %q, want string", existing.GoType)
		}
	})
}
