package repo

import (
	"strings"
	"testing"
)

func TestGenericRuntimeTemplateUsesRequestedPackage(t *testing.T) {
	content := genericRuntimeTemplate("generated")

	if !strings.Contains(content, "package generated\n") {
		t.Fatal("generic runtime template does not use the requested package")
	}
	if strings.Contains(content, "package runtime\n") {
		t.Fatal("generic runtime template leaked the source package")
	}
	if !strings.Contains(content, "func ApplyFilter(filter Filter) Applier") {
		t.Fatal("generic runtime template does not include generated adapters")
	}
}
