//go:build windows
// +build windows

package security

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
)

const (
	vmAccountName = `NT VIRTUAL MACHINE\\Virtual Machines`
	vmAccountSID  = "S-1-5-83-0"
)

// TestGrantVmGroupAccess verifies for the three case of a file, a directory,
// and a file in a directory that the appropriate ACEs are set, including
// inheritance in the second two examples. These are the expected ACES. Is
// verified by running icacls and comparing output.
//
// File:
// S-1-15-3-1024-2268835264-3721307629-241982045-173645152-1490879176-104643441-2915960892-1612460704:(R,W)
// S-1-5-83-1-3166535780-1122986932-343720105-43916321:(R,W)
//
// Directory:
// S-1-15-3-1024-2268835264-3721307629-241982045-173645152-1490879176-104643441-2915960892-1612460704:(OI)(CI)(R,W)
// S-1-5-83-1-3166535780-1122986932-343720105-43916321:(OI)(CI)(R,W)
//
// File in directory (inherited):
// S-1-15-3-1024-2268835264-3721307629-241982045-173645152-1490879176-104643441-2915960892-1612460704:(I)(R,W)
// S-1-5-83-1-3166535780-1122986932-343720105-43916321:(I)(R,W)

func TestGrantVmGroupAccessDefault(t *testing.T) {
	f1Path := filepath.Join(t.TempDir(), "gvmgafile")
	f, err := os.Create(f1Path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(f1Path)
	}()

	dir2 := t.TempDir()
	f2Path := filepath.Join(dir2, "find.txt")
	find, err := os.Create(f2Path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = find.Close()
		_ = os.Remove(f2Path)
	}()

	if err := GrantVmGroupAccess(f1Path); err != nil {
		t.Fatal(err)
	}

	if err := GrantVmGroupAccess(dir2); err != nil {
		t.Fatal(err)
	}

	verifyVMAccountDACLs(t,
		f1Path,
		[]string{`(R)`},
	)

	// Two items here:
	//  - One explicit read only.
	//  - Other applies to this folder, subfolders and files
	//      (OI): object inherit
	//      (CI): container inherit
	//      (IO): inherit only
	//      (GR): generic read
	//
	// In properties for the directory, advanced security settings, this will
	// show as a single line "Allow/Virtual Machines/Read/Inherited from none/This folder, subfolder and files
	verifyVMAccountDACLs(t,
		dir2,
		[]string{`(R)`, `(OI)(CI)(IO)(GR)`},
	)

	verifyVMAccountDACLs(t,
		f2Path,
		[]string{`(I)(R)`},
	)

}

func TestGrantVMGroupAccess_File_DesiredPermissions(t *testing.T) {
	type config struct {
		name                string
		desiredAccess       accessMask
		expectedPermissions []string
	}

	for _, cfg := range []config{
		{
			name:                "Read",
			desiredAccess:       AccessMaskRead,
			expectedPermissions: []string{`(R)`},
		},
		{
			name:                "Write",
			desiredAccess:       AccessMaskWrite,
			expectedPermissions: []string{`(W,Rc)`},
		},
		{
			name:                "Execute",
			desiredAccess:       AccessMaskExecute,
			expectedPermissions: []string{`(Rc,S,X,RA)`},
		},
		{
			name:                "ReadWrite",
			desiredAccess:       AccessMaskRead | AccessMaskWrite,
			expectedPermissions: []string{`(R,W)`},
		},
		{
			name:                "ReadExecute",
			desiredAccess:       AccessMaskRead | AccessMaskExecute,
			expectedPermissions: []string{`(RX)`},
		},
		{
			name:                "WriteExecute",
			desiredAccess:       AccessMaskWrite | AccessMaskExecute,
			expectedPermissions: []string{`(W,Rc,X,RA)`},
		},
		{
			name:                "ReadWriteExecute",
			desiredAccess:       AccessMaskRead | AccessMaskWrite | AccessMaskExecute,
			expectedPermissions: []string{`(RX,W)`},
		},
		{
			name:                "All",
			desiredAccess:       AccessMaskAll,
			expectedPermissions: []string{`(F)`},
		},
	} {
		t.Run(cfg.name, func(t *testing.T) {
			dir := t.TempDir()
			fd, err := os.Create(filepath.Join(dir, "test.txt"))
			if err != nil {
				t.Fatalf("failed to create temporary file: %s", err)
			}
			defer func() {
				_ = fd.Close()
				_ = os.Remove(fd.Name())
			}()

			if err := GrantVmGroupAccessWithMask(fd.Name(), cfg.desiredAccess); err != nil {
				t.Fatal(err)
			}
			verifyVMAccountDACLs(t, fd.Name(), cfg.expectedPermissions)
		})
	}
}

func TestGrantVMGroupAccess_Directory_Permissions(t *testing.T) {
	type config struct {
		name            string
		access          accessMask
		filePermissions []string
		dirPermissions  []string
	}

	for _, cfg := range []config{
		{
			name:            "Read",
			access:          AccessMaskRead,
			filePermissions: []string{`(I)(R)`},
			dirPermissions:  []string{`(R)`, `(OI)(CI)(IO)(GR)`},
		},
		{
			name:            "Write",
			access:          AccessMaskWrite,
			filePermissions: []string{`(I)(W,Rc)`},
			dirPermissions:  []string{`(W,Rc)`, `(OI)(CI)(IO)(GW)`},
		},
		{
			name:            "Execute",
			access:          AccessMaskExecute,
			filePermissions: []string{`(I)(Rc,S,X,RA)`},
			dirPermissions:  []string{`(Rc,S,X,RA)`, `(OI)(CI)(IO)(GE)`},
		},
		{
			name:            "ReadWrite",
			access:          AccessMaskRead | AccessMaskWrite,
			filePermissions: []string{`(I)(R,W)`},
			dirPermissions:  []string{`(R,W)`, `(OI)(CI)(IO)(GR,GW)`},
		},
		{
			name:            "ReadExecute",
			access:          AccessMaskRead | AccessMaskExecute,
			filePermissions: []string{`(I)(RX)`},
			dirPermissions:  []string{`(RX)`, `(OI)(CI)(IO)(GR,GE)`},
		},
		{
			name:            "WriteExecute",
			access:          AccessMaskWrite | AccessMaskExecute,
			filePermissions: []string{`(I)(W,Rc,X,RA)`},
			dirPermissions:  []string{`(W,Rc,X,RA)`, `(OI)(CI)(IO)(GW,GE)`},
		},
		{
			name:            "ReadWriteExecute",
			access:          AccessMaskRead | AccessMaskWrite | AccessMaskExecute,
			filePermissions: []string{`(I)(RX,W)`},
			dirPermissions:  []string{`(RX,W)`, `(OI)(CI)(IO)(GR,GW,GE)`},
		},
		{
			name:            "All",
			access:          AccessMaskAll,
			filePermissions: []string{`(I)(F)`},
			dirPermissions:  []string{`(F)`, `(OI)(CI)(IO)(F)`},
		}} {
		t.Run(cfg.name, func(t *testing.T) {
			dir := t.TempDir()
			fd, err := os.Create(filepath.Join(dir, "test.txt"))
			if err != nil {
				t.Fatalf("failed to create temporary file: %s", err)
			}
			defer func() {
				_ = fd.Close()
				_ = os.Remove(fd.Name())
			}()

			if err := GrantVmGroupAccessWithMask(dir, cfg.access); err != nil {
				t.Fatal(err)
			}
			verifyVMAccountDACLs(t, dir, cfg.dirPermissions)
			verifyVMAccountDACLs(t, fd.Name(), cfg.filePermissions)
		})
	}
}

func TestGrantVmGroupAccess_Invalid_AccessMask(t *testing.T) {
	for _, access := range []accessMask{
		0,          // no bits set
		1,          // invalid bit set
		0x02000001, // invalid extra bit set
	} {
		t.Run(fmt.Sprintf("AccessMask_0x%x", access), func(t *testing.T) {
			dir := t.TempDir()
			fd, err := os.Create(filepath.Join(dir, "test.txt"))
			if err != nil {
				t.Fatalf("failed to create temporary file: %s", err)
			}
			defer func() {
				_ = fd.Close()
				_ = os.Remove(fd.Name())
			}()

			if err := GrantVmGroupAccessWithMask(fd.Name(), access); err == nil {
				t.Fatalf("expected an error for mask: %x", access)
			}
		})
	}
}

func verifyVMAccountDACLs(t *testing.T, name string, permissions []string) {
	cmd := exec.Command("icacls", name)
	outb, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	out := string(outb)

	for _, p := range permissions {
		// Avoid '(' and ')' being part of match groups
		p = regexp.QuoteMeta(p)

		nameToCheck := vmAccountName + ":" + p
		sidToCheck := vmAccountSID + ":" + p

		rxName := regexp.MustCompile(nameToCheck)
		rxSID := regexp.MustCompile(sidToCheck)

		matchesName := rxName.FindAllStringIndex(out, -1)
		matchesSID := rxSID.FindAllStringIndex(out, -1)

		if len(matchesName) != 1 && len(matchesSID) != 1 {
			t.Fatalf("expected one match for %s or %s\n%s\n", nameToCheck, sidToCheck, out)
		}
	}
}
