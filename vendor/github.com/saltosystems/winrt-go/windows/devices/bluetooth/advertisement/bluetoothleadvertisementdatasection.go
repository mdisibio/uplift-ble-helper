// Code generated by winrt-go-gen. DO NOT EDIT.

//go:build windows

//nolint:all
package advertisement

import (
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
)

const SignatureBluetoothLEAdvertisementDataSection string = "rc(Windows.Devices.Bluetooth.Advertisement.BluetoothLEAdvertisementDataSection;{d7213314-3a43-40f9-b6f0-92bfefc34ae3})"

type BluetoothLEAdvertisementDataSection struct {
	ole.IUnknown
}

func NewBluetoothLEAdvertisementDataSection() (*BluetoothLEAdvertisementDataSection, error) {
	inspectable, err := ole.RoActivateInstance("Windows.Devices.Bluetooth.Advertisement.BluetoothLEAdvertisementDataSection")
	if err != nil {
		return nil, err
	}
	return (*BluetoothLEAdvertisementDataSection)(unsafe.Pointer(inspectable)), nil
}

func (impl *BluetoothLEAdvertisementDataSection) GetDataType() (uint8, error) {
	itf := impl.MustQueryInterface(ole.NewGUID(GUIDiBluetoothLEAdvertisementDataSection))
	defer itf.Release()
	v := (*iBluetoothLEAdvertisementDataSection)(unsafe.Pointer(itf))
	return v.GetDataType()
}

const GUIDiBluetoothLEAdvertisementDataSection string = "d7213314-3a43-40f9-b6f0-92bfefc34ae3"
const SignatureiBluetoothLEAdvertisementDataSection string = "{d7213314-3a43-40f9-b6f0-92bfefc34ae3}"

type iBluetoothLEAdvertisementDataSection struct {
	ole.IInspectable
}

type iBluetoothLEAdvertisementDataSectionVtbl struct {
	ole.IInspectableVtbl

	GetDataType uintptr
	SetDataType uintptr
	GetData     uintptr
	SetData     uintptr
}

func (v *iBluetoothLEAdvertisementDataSection) VTable() *iBluetoothLEAdvertisementDataSectionVtbl {
	return (*iBluetoothLEAdvertisementDataSectionVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iBluetoothLEAdvertisementDataSection) GetDataType() (uint8, error) {
	var out uint8
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetDataType,
		uintptr(unsafe.Pointer(v)),    // this
		uintptr(unsafe.Pointer(&out)), // out uint8
	)

	if hr != 0 {
		return 0, ole.NewError(hr)
	}

	return out, nil
}
