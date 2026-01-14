package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultLabels(t *testing.T) {
	ftime := func() time.Time {
		return time.Time{}
	}
	Now(ftime)
	tests := []struct {
		name      string
		fsOpts    options
		expResult map[string]string
	}{
		{
			name: "driver loaded, all labels found",
			fsOpts: options{
				hlVersion:     "1.17.0-987abcd",
				vendor:        "vendor",
				productName:   "HLS2 B81.04B01.0013",
				productSerial: "W-M-10-00005U",
				revision:      "0x01",
				fwOS:          "Zephyr 2.7.2-hl-gaudi2-1.16.0-fw-50.0.0-sec-9 (May  5 2024 - 08:09:32)",
				device:        "0x1020",
				devType:       "GAUDI2",
			},
			expResult: map[string]string{
				TimestampLabel:           fmt.Sprintf("%d", ftime().Unix()),
				DeviceCountLabel:         "1",
				DeviceFamilyLabel:        "gaudi",
				SysVendorLabel:           "vendor",
				ProductNameLabel:         "HLS2_B81.04B01.0013",
				ProductSerialLabel:       "W-M-10-00005U",
				DeviceRevisionLabel:      "01",
				DeviceIDLabel:            "1020",
				DriverHashLabel:          "987abcd",
				DriverVersionLabel:       "1.17.0",
				FirmwareVersionLabel:     "hl-gaudi2-1.16.0-fw-50.0.0",
				FirmwareOSLabel:          "Zephyr",
				DeviceTypeLabel:          "GAUDI2",
				DistroTypeLabel:          "ubuntu_22.04",
				DistroOsTreeVersionLabel: "",
				KernelVersionLabel:       "5.15.0-107-generic",
			},
		},
		{
			name: "driver not loaded firware related should by empty",
			fsOpts: options{
				hlVersion:     "",
				vendor:        "vendor",
				productName:   "HLS2 B81.04B01.0013",
				productSerial: "W-M-10-00005U",
				revision:      "0x01",
				fwOS:          "Zephyr 2.7.2-hl-gaudi2-1.16.0-fw-50.0.0-sec-9 (May  5 2024 - 08:09:32)",
				device:        "0x1020",
				devType:       "GAUDI2",
			},
			expResult: map[string]string{
				TimestampLabel:           fmt.Sprintf("%d", ftime().Unix()),
				DeviceCountLabel:         "1",
				DeviceFamilyLabel:        "gaudi",
				SysVendorLabel:           "vendor",
				ProductNameLabel:         "HLS2_B81.04B01.0013",
				ProductSerialLabel:       "W-M-10-00005U",
				DeviceRevisionLabel:      "01",
				DeviceIDLabel:            "1020",
				DriverHashLabel:          "",
				DriverVersionLabel:       "",
				FirmwareVersionLabel:     "",
				FirmwareOSLabel:          "",
				DeviceTypeLabel:          "",
				DistroTypeLabel:          "ubuntu_22.04",
				DistroOsTreeVersionLabel: "",
				KernelVersionLabel:       "5.15.0-107-generic",
			},
		},
		{
			name: "driver not gaudi 1 - no fw_os_ver content",
			fsOpts: options{
				hlVersion:     "",
				vendor:        "vendor",
				productName:   "HLS2 B81.04B01.0013",
				productSerial: "W-M-10-00005U",
				revision:      "0x01",
				fwOS:          "",
				device:        "0x1020",
				devType:       "GAUDI2",
			},
			expResult: map[string]string{
				TimestampLabel:           fmt.Sprintf("%d", ftime().Unix()),
				DeviceCountLabel:         "1",
				DeviceFamilyLabel:        "gaudi",
				SysVendorLabel:           "vendor",
				ProductNameLabel:         "HLS2_B81.04B01.0013",
				ProductSerialLabel:       "W-M-10-00005U",
				DeviceRevisionLabel:      "01",
				DeviceIDLabel:            "1020",
				DriverHashLabel:          "",
				DriverVersionLabel:       "",
				FirmwareVersionLabel:     "",
				FirmwareOSLabel:          "",
				DeviceTypeLabel:          "",
				DistroTypeLabel:          "ubuntu_22.04",
				DistroOsTreeVersionLabel: "",
				KernelVersionLabel:       "5.15.0-107-generic",
			},
		},
		{
			name: "two words device type",
			fsOpts: options{
				hlVersion:     "1.17.0-987abcd",
				vendor:        "vendor",
				productName:   "HLS2 B81.04B01.0013",
				productSerial: "W-M-10-00005U",
				revision:      "0x01",
				fwOS:          "Zephyr 2.7.2-hl-gaudi2-1.16.0-fw-50.0.0-sec-9 (May  5 2024 - 08:09:32)",
				device:        "0x1020",
				devType:       "GAUDI HL2000M",
			},
			expResult: map[string]string{
				TimestampLabel:           fmt.Sprintf("%d", ftime().Unix()),
				DeviceCountLabel:         "1",
				DeviceFamilyLabel:        "gaudi",
				SysVendorLabel:           "vendor",
				ProductNameLabel:         "HLS2_B81.04B01.0013",
				ProductSerialLabel:       "W-M-10-00005U",
				DeviceRevisionLabel:      "01",
				DeviceIDLabel:            "1020",
				DriverHashLabel:          "987abcd",
				DriverVersionLabel:       "1.17.0",
				FirmwareVersionLabel:     "hl-gaudi2-1.16.0-fw-50.0.0",
				FirmwareOSLabel:          "Zephyr",
				DeviceTypeLabel:          "GAUDI_HL2000M",
				DistroTypeLabel:          "ubuntu_22.04",
				DistroOsTreeVersionLabel: "",
				KernelVersionLabel:       "5.15.0-107-generic",
			},
		},
		{
			name: "invalid entries for labels",
			fsOpts: options{
				hlVersion:     "1.17.0-987abcd",
				vendor:        "01234567890123456789012345678901234567890123456789012345678901234",
				productName:   "???__aabb__ccdd__;;;",
				productSerial: "Standard_PC_(i440FX_+PIIX1996)",
				revision:      "0x01",
				fwOS:          "Zephyr 2.7.2-hl-gaudi2-1.16.0-fw-50.0.0-sec-9 (May  5 2024 - 08:09:32)",
				device:        "0x1020",
				devType:       "______________________________________________________________________",
			},
			expResult: map[string]string{
				TimestampLabel:           fmt.Sprintf("%d", ftime().Unix()),
				DeviceCountLabel:         "1",
				DeviceFamilyLabel:        "gaudi",
				SysVendorLabel:           "012345678901234567890123456789012345678901234567890123456789012",
				ProductNameLabel:         "aabb__ccdd",
				ProductSerialLabel:       "Standard_PC__i440FX__PIIX1996",
				DeviceRevisionLabel:      "01",
				DeviceIDLabel:            "1020",
				DriverHashLabel:          "987abcd",
				DriverVersionLabel:       "1.17.0",
				FirmwareVersionLabel:     "hl-gaudi2-1.16.0-fw-50.0.0",
				FirmwareOSLabel:          "Zephyr",
				DeviceTypeLabel:          "",
				DistroTypeLabel:          "ubuntu_22.04",
				DistroOsTreeVersionLabel: "",
				KernelVersionLabel:       "5.15.0-107-generic",
			},
		},
		{
			name: "invalid entries",
			fsOpts: options{
				hlVersion:     "1.17.0-987abcd",
				vendor:        "01234567890123456789012345678901234567890123456789012345678901234",
				productName:   "???__aabb__ccdd__;;;.",
				productSerial: "Dell_Inc.",
				revision:      "0x01",
				fwOS:          "Zephyr 2.7.2-hl-gaudi2-1.16.0-fw-50.0.0-sec-9 (May  5 2024 - 08:09:32)",
				device:        "0x1020",
				devType:       "______________________________________________________________________",
			},
			expResult: map[string]string{
				TimestampLabel:           fmt.Sprintf("%d", ftime().Unix()),
				DeviceCountLabel:         "1",
				DeviceFamilyLabel:        "gaudi",
				SysVendorLabel:           "012345678901234567890123456789012345678901234567890123456789012",
				ProductNameLabel:         "aabb__ccdd",
				ProductSerialLabel:       "Dell_Inc",
				DeviceRevisionLabel:      "01",
				DeviceIDLabel:            "1020",
				DriverHashLabel:          "987abcd",
				DriverVersionLabel:       "1.17.0",
				FirmwareVersionLabel:     "hl-gaudi2-1.16.0-fw-50.0.0",
				FirmwareOSLabel:          "Zephyr",
				DeviceTypeLabel:          "",
				DistroTypeLabel:          "ubuntu_22.04",
				DistroOsTreeVersionLabel: "",
				KernelVersionLabel:       "5.15.0-107-generic",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				KernelVersionFunction = KernelVersion
			})
			KernelVersionFunction = func() string { return "5.15.0-107-generic" }
			// Create the testing filesystem
			tdir, err := os.MkdirTemp("", "hfd-*")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = os.RemoveAll(tdir)
			})

			prepareTreeStructure(t, tdir, tt.fsOpts)

			// Set the root FS for the temporary dir created
			TestFS(tdir)

			got, err := DefaultLabels()
			if err != nil {
				t.Fatal(err)
			}

			// Assert results
			for k, v := range got {
				gv, ok := tt.expResult[k]
				if !ok {
					t.Logf("%q is %q", k, v)
					t.Errorf("found key %q, but it is not expected", k)
					continue
				}
				if v != gv {
					t.Errorf("key=%s, expected %q, got %q", k, v, gv)
				}
			}
		})
	}
}

type options struct {
	hlVersion                  string
	vendor                     string
	productName, productSerial string
	revision                   string
	fwOS                       string
	device, devType            string
}

func prepareTreeStructure(t *testing.T, tmpDir string, opts options) {
	{
		modFS := filepath.Join(tmpDir, "sys/module/habanalabs")
		t.Logf("Creating tree for %s\n", modFS)

		err := os.MkdirAll(modFS, 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(modFS+"/version", []byte(opts.hlVersion), 0744)
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		pciDevFS := filepath.Join(tmpDir, "sys/bus/pci/devices/0000:33:00.0")
		t.Logf("Creating tree for %s\n", pciDevFS)

		err := os.MkdirAll(tmpDir+"/sys/bus/pci/devices/0000:33:00.0", 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(pciDevFS+"/vendor", []byte("0x1da3"), 0744)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(pciDevFS+"/device", []byte(opts.device), 0744)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(pciDevFS+"/revision", []byte(opts.revision), 0744)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Created all files", pciDevFS)
	}

	{
		classFS := filepath.Join(tmpDir, "sys/class/accel/accel0/device")
		t.Logf("Creating tree for %s\n", classFS)

		err := os.MkdirAll(classFS+"/device", 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(classFS+"/fw_os_ver", []byte(opts.fwOS), 0744)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(classFS+"/device_type", []byte(opts.devType), 0744)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Created all files", classFS)
	}

	{
		dmiFS := filepath.Join(tmpDir, "/sys/class/dmi/id")
		t.Logf("Creating tree for %s\n", dmiFS)

		err := os.MkdirAll(dmiFS, 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(dmiFS+"/sys_vendor", []byte(opts.vendor), 0744)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(dmiFS+"/product_name", []byte(opts.productName), 0744)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(dmiFS+"/product_serial", []byte(opts.productSerial), 0744)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Created all files", dmiFS)
	}
	{
		osFS := filepath.Join(tmpDir, "usr/lib")
		t.Logf("Creating tree for %s\n", osFS)
		err := os.MkdirAll(osFS, 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(osFS+"/os-release", []byte("VERSION_ID=\"22.04\"\nID=\"ubuntu\""), 0744)
		if err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(osFS + "/os-release")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("os-release file content: %s", content)
		t.Log("Created all files", osFS)
		t.Log(os.DirFS(osFS))
	}
}
