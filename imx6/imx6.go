// NXP i.MX6UL/i.MX6ULL/i.MX6Q support
// https://github.com/f-secure-foundry/tamago
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// Package imx6 provides support to Go bare metal unikernels written using the
// TamaGo framework.
//
// The package implements initialization and drivers for specific NXP i.MX6
// System-on-Chip (SoC) peripherals, adopting, where indicated, the following
// reference specifications:
//   * IMX6ULLRM  - i.MX 6ULL Applications Processor Reference Manual                - Rev 1      2017/11
//   * IMX6FG     - i.MX 6 Series Firmware Guide                                     - Rev 0      2012/11
//   * IMX6ULLCEC - i.MX6ULL Data Sheet                                              - Rev 1.2    2017/11
//   * MCIMX28RM  - i.MX28 Applications Processor Reference Manual                   - Rev 2      2013/08
//   * SD-PL-7.10 - SD Specifications Part 1 Physical Layer Simplified Specification - 7.10       2020/03/25
//   * JESD84-B51 - Embedded Multi-Media Card (e•MMC) Electrical Standard (5.1)      - JESD84-B51 2015/02
//   * USB2.0     - USB Specification Revision 2.0                                   - 2.0        2000/04/27
//
// This package is only meant to be used with `GOOS=tamago GOARCH=arm` as
// supported by the TamaGo framework for bare metal Go on ARM SoCs, see
// https://github.com/f-secure-foundry/tamago.
package imx6

import (
	"encoding/binary"
	_ "unsafe"

	"github.com/f-secure-foundry/tamago/arm"
	"github.com/f-secure-foundry/tamago/internal/reg"
)

// Identification registers
const (
	WDOG1_WCR          uint32 = 0x020bc000
	OCOTP_CFG0         uint32 = 0x021bc410
	OCOTP_CFG1         uint32 = 0x021bc420
	USB_ANALOG_DIGPROG uint32 = 0x020c8260
)

// Reset registers
const (
	SRC_SCR               uint32 = 0x020d8000
	SCR_WARM_RESET_ENABLE        = 0
)

// Timer register
// (p178, Table 2-3, IMX6ULLRM)
const SYS_CNT_BASE uint32 = 0x021dc000

// i.MX processor families
const (
	IMX6Q   = 0x63
	IMX6UL  = 0x64
	IMX6ULL = 0x65
)

// Processor family
var Family uint32

// Flag for native or emulated processor
var Native bool

// ARM processor instance
var ARM = &arm.CPU{}

//go:linkname nanotime1 runtime.nanotime1
func nanotime1() int64 {
	return int64(ARM.TimerFn() * ARM.TimerMultiplier)
}

// hwinit takes care of the lower level SoC initialization triggered early in
// runtime setup, care must be taken to ensure that no heap allocation is
// performed (e.g. defer is not possible).
//go:linkname hwinit runtime.hwinit
func hwinit() {
	ARM.Init()
	ARM.EnableVFP()
	ARM.CacheEnable()

	_, fam, revMajor, revMinor := SiliconVersion()
	Family = fam

	if revMajor != 0 || revMinor != 0 {
		Native = true
	}

	// initialize console
	UART2.init(UART2_BASE, UART_DEFAULT_BAUDRATE)

	switch Family {
	case IMX6Q:
		ARM.InitGlobalTimers()
	case IMX6UL, IMX6ULL:
		if !Native {
			// use QEMU fixed CNTFRQ value (62.5MHz)
			ARM.InitGenericTimers(SYS_CNT_BASE, 62500000)
		} else {
			// U-Boot value for i.MX6 family (8.0MHz)
			ARM.InitGenericTimers(SYS_CNT_BASE, 8000000)
		}
	default:
		ARM.InitGlobalTimers()
	}
}

//go:linkname initRNG runtime.initRNG
func initRNG() {
	if Family == IMX6ULL && Native {
		RNGB.Init()
		getRandomDataFn = RNGB.getRandomData
	} else {
		getRandomDataFn = getLCGData
	}
}

// SiliconVersion returns the SoC silicon version information
// (p3945, 57.4.11 Chip Silicon Version (USB_ANALOG_DIGPROG), IMX6ULLRM).
func SiliconVersion() (sv, family, revMajor, revMinor uint32) {
	sv = reg.Read(USB_ANALOG_DIGPROG)

	family = (sv >> 16) & 0xff
	revMajor = (sv >> 8) & 0xff
	revMinor = sv & 0xff

	return
}

// UniqueID returns the NXP SoC Device Unique 64-bit ID.
func UniqueID() (uid [8]byte) {
	cfg0 := reg.Read(OCOTP_CFG0)
	cfg1 := reg.Read(OCOTP_CFG1)

	binary.LittleEndian.PutUint32(uid[0:4], cfg0)
	binary.LittleEndian.PutUint32(uid[4:8], cfg1)

	return
}

// Model returns the SoC model name.
func Model() (model string) {
	switch Family {
	case IMX6Q:
		model = "i.MX6Q"
	case IMX6UL:
		model = "i.MX6UL"
	case IMX6ULL:
		model = "i.MX6ULL"
	default:
		model = "unknown"
	}

	return
}

// Reboot resets the watchdog timer causing the SoC to restart.
func Reboot() {
	reg.Clear(SRC_SCR, SCR_WARM_RESET_ENABLE)
	// WDOG1_WCR is a 16-bit register, 32-bit access should be avoided
	reg.Write16(WDOG1_WCR, 0)
}
