// Package collectors provides functions retrieving data from the system
// host about Habana PCI cards and the bare-metal itself.
//
// It is designed to be used and match with Kubernetes so the labels
// and values are in a compatible format.
package collector

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	HabanaVendor = "1da3"
	DeviceFamily = "gaudi"

	HabanaPrefix             = "habana.ai/"
	TimestampLabel           = HabanaPrefix + "hfd.timestamp"
	SysVendorLabel           = HabanaPrefix + "sys.vendor"
	DriverVersionLabel       = HabanaPrefix + "driver.version"
	DriverHashLabel          = HabanaPrefix + "driver.hash"
	DeviceFamilyLabel        = HabanaPrefix + "device.family"
	DeviceCountLabel         = HabanaPrefix + "device.count"
	DeviceNameLabel          = HabanaPrefix + "device.name"
	DeviceRevisionLabel      = HabanaPrefix + "device.revision"
	DeviceIDLabel            = HabanaPrefix + "device.id"
	DeviceTypeLabel          = HabanaPrefix + "device.type"
	DistroTypeLabel          = HabanaPrefix + "distro.type"
	DistroOsTreeVersionLabel = HabanaPrefix + "distro.ostree-version"
	FirmwareVersionLabel     = HabanaPrefix + "fw.version"
	FirmwareOSLabel          = HabanaPrefix + "fw.os"
	KernelVersionLabel       = HabanaPrefix + "kernel.version"
	ProductSerialLabel       = HabanaPrefix + "product.serial"
	ProductNameLabel         = HabanaPrefix + "product.name"
)

var (
	fwVerExtractor = regexp.MustCompile(`hl-gaudi\d-(\d+\.\d+\.\d+)-fw-(\d+\.\d+\.\d+)`)

	rootPrefix  = "/"
	sysfsDir    = rootPrefix + "sys"
	accelFSRoot = sysfsDir + "/class/accel"
	pciAttrs    = []string{"device", "revision"}
	dmiAttrs    = []string{"product_name", "product_serial", "sys_vendor"}

	now                   = time.Now
	KernelVersionFunction = KernelVersion
)

func init() {
	if os.Getenv("HFD_ROOT_PREFIX") != "" {
		rootPrefix = os.Getenv("HFD_ROOT_PREFIX")
	}
}

// SystemDriverVersion returns the loaded kernel version of `habanalabs` module.
func SystemDriverVersion() (string, error) {
	driver, err := os.ReadFile(sysfsDir + "/module/habanalabs/version")
	if err != nil {
		return "", fmt.Errorf("file reading error %w", err)
	}
	return strings.TrimSpace(string(driver)), nil
}

// ProductName returns the `product_name` as it appears in the DMI table.
func ProductName() (string, error) {
	attrs, err := DmiAttributes()
	if err != nil {
		return "", fmt.Errorf("product name: %w", err)
	}
	return attrs["product_name"], nil
}

// PCIInfo hold the related shared data of the devices on the machine, an their count.
type PCIInfo struct {
	Count    int
	DeviceID string
	Revision string
}

// PCIDeviceInformation returns `PCIInfo` for the machine.
func PCIDeviceInformation() (PCIInfo, error) {
	sysfsBasePath := path.Join(sysfsDir, "bus/pci/devices")

	pciDevices, err := os.ReadDir(sysfsBasePath)
	if err != nil {
		return PCIInfo{}, err
	}

	var devInfo PCIInfo
	for _, device := range pciDevices {
		devPath := path.Join(sysfsBasePath, device.Name())
		// We care only about our devices
		if !isHabanaPCI(devPath) {
			continue
		}

		// Getting information from one device is enough
		if devInfo.Count == 0 {
			info, err := readPCIDevInfo(devPath)
			if err != nil {
				continue
			}
			devInfo = PCIInfo{
				DeviceID: info["device"],
				Revision: info["revision"],
			}
		}
		devInfo.Count++
	}

	return devInfo, nil
}

// FWInfo hold the information retrieved from the device's fw_os_ver.
type FWInfo struct {
	// Firmware OS version
	OS string
	// Firmware Version
	Version string
	// DeviceType is the phonetic name of the device
	DeviceType string
}

// FWVersion returns the firmware version for a given device
func FWVersion() (FWInfo, error) {
	matches, err := filepath.Glob(accelFSRoot + "/accel?")
	if err != nil {
		panic("check the glob syntax")
	}
	if len(matches) == 0 {
		return FWInfo{}, fmt.Errorf("no devices found")
	}

	accelDev := matches[0]

	// In Gaudi 1 this value can be empty
	os, err := cleanRead(fmt.Sprintf("%s/device/fw_os_ver", accelDev))
	if err != nil {
		return FWInfo{}, err
	}

	// Extract version and format due to labels limitations
	version := "na"
	matches = fwVerExtractor.FindStringSubmatch(os)
	if len(matches) > 0 {
		version = matches[0]
		os = strings.Fields(os)[0]
	}

	devType, err := cleanRead(fmt.Sprintf("%s/device/device_type", accelDev))
	if err != nil {
		return FWInfo{}, err
	}

	return FWInfo{
		OS:         os,
		Version:    version,
		DeviceType: devType,
	}, nil
}

// DeviceName returns the device name as specified in the `device_name` sysfs based on device PCI id.
func DeviceName(deviceID string) string {
	gaudi1 := []string{"1000", "1001", "1010", "1011"}
	gaudi2 := []string{"1020", "1030", "1060", "1061", "1062"}
	gaudi3 := []string{"1060", "1061", "1062"}

	switch {
	case slices.Contains(gaudi1, deviceID):
		return "gaudi1"
	case slices.Contains(gaudi2, deviceID):
		return "gaudi2"
	case slices.Contains(gaudi3, deviceID):
		return "gaudi3"
	default:
		return ""
	}
}

// Vendor returns the vendor manufacturer of the bare-metal server.
func Vendor() (string, error) {
	attrs, err := DmiAttributes()
	if err != nil {
		return "", fmt.Errorf("getting vendor: %w", err)
	}
	return attrs["vendor"], nil
}

// DmiAttributes returns collected data from the Desktop Management Interface.
// Returns: "product_name", "product_serial", "sys_vendor"
func DmiAttributes() (map[string]string, error) {
	dmiFsDir := path.Join(sysfsDir, "class/dmi/id")

	attrs := make(map[string]string)
	for _, attr := range dmiAttrs {
		attrVal, err := cleanRead(path.Join(dmiFsDir, attr))
		if err != nil {
			return nil, fmt.Errorf("reading dmi attribute: %w", err)
		}
		if attr == "sys_vendor" && strings.Contains(attrVal, "To be filled by") {
			attrVal = "missing_vendor"
		}
		attrs[attr] = attrVal
	}

	return attrs, nil
}

// DefaultLabels returns a map of all the available collected info as string "labels" and their values,
// formatted correctly to be used as kubernetes labels. Labels' values that it couldn't collect, will be empty.
//
// For more control, you can use the separate functions in the package.
func DefaultLabels() (map[string]string, error) {
	// Initial values, always exist
	labels := map[string]string{
		TimestampLabel:    fmt.Sprintf("%d", now().Unix()),
		DeviceFamilyLabel: DeviceFamily,
	}

	dmiValues, err := DmiAttributes()
	if err != nil {
		return nil, fmt.Errorf("default labels: %w", err)
	}
	labels[SysVendorLabel] = sanitizeLabelValue(dmiValues["sys_vendor"])
	labels[ProductNameLabel] = sanitizeLabelValue(dmiValues["product_name"])
	labels[ProductSerialLabel] = sanitizeLabelValue(dmiValues["product_serial"])

	devInfo, err := PCIDeviceInformation()
	if err != nil {
		return nil, fmt.Errorf("collecting pci device info: %w", err)
	}
	labels[DeviceCountLabel] = fmt.Sprintf("%d", devInfo.Count)
	labels[DeviceRevisionLabel] = devInfo.Revision
	labels[DeviceIDLabel] = devInfo.DeviceID
	distro, err := DistroInfo()
	if err != nil {
		return nil, fmt.Errorf("collecting distro info: %w", err)
	}
	labels[DistroTypeLabel] = sanitizeLabelValue(distro.Name + "_" + distro.Version)
	labels[DistroOsTreeVersionLabel] = sanitizeLabelValue(distro.OsTreeVersion)
	driverVersion, err := SystemDriverVersion()
	if err != nil {
		slog.Error("Error getting system driver version", "error", err)
	}
	labels[KernelVersionLabel] = sanitizeLabelValue(KernelVersionFunction())
	if devInfo.Count == 0 {
		labels[DeviceFamilyLabel] = sanitizeLabelValue("generic")
	}
	switch {
	// Driver  is loaded, can collect related information.
	case driverVersion != "":
		release, hash, _ := strings.Cut(driverVersion, "-")
		labels[DriverVersionLabel] = sanitizeLabelValue(release)
		labels[DriverHashLabel] = sanitizeLabelValue(hash)

		fwInfo, err := FWVersion()
		if err == nil { // IF NO ERROR populate labels
			labels[FirmwareVersionLabel] = sanitizeLabelValue(fwInfo.Version)
			labels[FirmwareOSLabel] = sanitizeLabelValue(fwInfo.OS)
			labels[DeviceTypeLabel] = sanitizeLabelValue(fwInfo.DeviceType)
		}
	default:
		labels[DriverVersionLabel] = ""
		labels[DriverHashLabel] = ""
		labels[FirmwareVersionLabel] = ""
		labels[FirmwareOSLabel] = ""
		labels[DeviceTypeLabel] = ""

	}

	return labels, nil
}

// Define the function at the package level
func isInvalidChar(c string) bool {
	chars := []string{"_", ".", "-"}
	return slices.Contains(chars, c)
}

func sanitizeLabelValue(val string) string {
	if len(val) == 0 {
		return val
	}

	// Regexp for accepted characters
	re := regexp.MustCompile("[^A-Za-z0-9_.-]")

	// Replace all non-accepted characters with underscore
	result := re.ReplaceAllString(val, "_")

	// Remove leading and trailing invalid characters
	for len(result) > 0 && isInvalidChar(string(result[0])) {
		result = result[1:]
	}

	for len(result) > 0 && isInvalidChar(string(result[len(result)-1])) {
		result = result[:len(result)-1]
	}

	// Limit label to 63 characters
	if len(result) > 63 {
		result = result[:63]
	}

	return result
}

func readPCIDevInfo(devPath string) (map[string]string, error) {
	attrs := make(map[string]string)
	for _, attr := range pciAttrs {
		attrVal, err := readSinglePCIAttr(devPath, attr)
		if err != nil {
			return nil, fmt.Errorf("reading device %s: %w", attr, err)
		}
		attrs[attr] = attrVal
	}
	return attrs, nil
}

func readSinglePCIAttr(devPath, attrName string) (string, error) {
	attrVal, err := cleanRead(path.Join(devPath, attrName))
	if err != nil {
		return "", fmt.Errorf("reading attribute %s: %w", attrName, err)
	}

	if attrName == "class" && len(attrVal) > 4 {
		attrVal = attrVal[0:4]
	}

	return attrVal, nil
}

func isHabanaPCI(devPath string) bool {
	attrVal, err := readSinglePCIAttr(devPath, "vendor")
	if err != nil {
		slog.Error("Error reading vendor attribute", "error", err, "path", devPath)
		return false
	}
	return attrVal == HabanaVendor
}

func cleanRead(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.TrimPrefix(string(data), "0x")), nil
}

type Distro struct {
	Name          string
	Version       string
	OsTreeVersion string
}

func DistroInfo() (Distro, error) {
	distroFilePaths := []string{
		path.Join(rootPrefix, "usr/lib/os-release"),
		path.Join(rootPrefix, "etc/os-release"),
	}

	var file *os.File
	var err error

	for _, filePath := range distroFilePaths {
		file, err = os.Open(filePath)
		if err != nil {
			log.Printf("Failed to open %s: %v", filePath, err)
			continue
		}

		// Read a few lines to check for validity
		scanner := bufio.NewScanner(file)
		isValid := false
		for i := 0; i < 10 && scanner.Scan(); i++ { // Read up to 10 lines
			line := scanner.Text()
			if strings.HasPrefix(line, "ID=") || strings.HasPrefix(line, "VERSION_ID=") || strings.HasPrefix(line, "RHEL_VERSION=") {
				isValid = true
				break
			}
		}
		// Reset file pointer to the beginning
		_, err = file.Seek(0, 0)
		if err != nil {
			err = file.Close()
			if err != nil {
				log.Printf("Failed to close file for %s: %v", filePath, err)
			}
			return Distro{}, err
		}

		if isValid {
			break
		} else {
			err := file.Close()
			if err != nil {
				log.Printf("Failed to close file %s: %v", filePath, err)
			}
			file = nil
		}
	}

	if file == nil {
		return Distro{}, err
	}
	defer file.Close()

	var info Distro
	var rhelVersion string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		key, value := TrimOsReleaseLine(line)
		switch key {
		case "ID":
			info.Name = value
		case "VERSION_ID":
			info.Version = value
		case "RHEL_VERSION":
			rhelVersion = value
		case "OSTREE_VERSION":
			info.OsTreeVersion = value
		}
	}

	if info.Name == "rhcos" {
		info.Name = "rhel"
		if rhelVersion != "" {
			info.Version = rhelVersion
		}
	}

	if err := scanner.Err(); err != nil {
		return Distro{}, err
	}

	return info, nil
}

func TrimOsReleaseLine(line string) (string, string) {
	var key, value string
	splittedLine := strings.Split(line, "=")

	// Ensure the line is in format of key=value to avoid overflow
	if len(splittedLine) == 2 {
		key = splittedLine[0]
		value = splittedLine[1]
		value = strings.Trim(value, `"`)
	}
	return key, value
}

func KernelVersion() string {
	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		// handle error
		return ""
	}
	return string(output)
}
