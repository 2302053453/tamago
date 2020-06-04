// NXP i.MX6 UART driver
// https://github.com/f-secure-foundry/tamago
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package imx6

import (
	"sync"

	"github.com/f-secure-foundry/tamago/internal/bits"
	"github.com/f-secure-foundry/tamago/internal/reg"
)

// UART registers
const (
	UART_DEFAULT_BAUDRATE = 115200
	ESC                   = 0x1b

	// p2315, 45.15 UART Memory Map/Register Definition, IMX6ULLRM

	// i.MX 6UltraLite (G0, G1, G2, G3, G4)
	// i.MX 6ULL (Y0, Y1, Y2)
	// i.MX 6ULZ (Z0)
	UART1_BASE uint32 = 0x02020000
	UART2_BASE uint32 = 0x021e8000
	UART3_BASE uint32 = 0x021ec000
	UART4_BASE uint32 = 0x021f0000

	// i.MX 6UltraLite (G1, G2, G3, G4)
	// i.MX 6ULL (Y1, Y2)
	UART5_BASE uint32 = 0x021f4000
	UART6_BASE uint32 = 0x021fc000
	UART7_BASE uint32 = 0x02018000
	UART8_BASE uint32 = 0x02024000

	UARTx_URXD   uint32 = 0x0000
	URXD_CHARRDY        = 15
	URXD_ERR            = 14
	URXD_OVRRUN         = 13
	URXD_FRMERR         = 12
	URXD_BRK            = 11
	URXD_PRERR          = 10
	URXD_RX_DATA        = 0

	UARTx_UTXD   uint32 = 0x0040
	UTXD_TX_DATA        = 0

	UARTx_UCR1    uint32 = 0x0080
	UCR1_ADEN            = 15
	UCR1_ADBR            = 14
	UCR1_TRDYEN          = 13
	UCR1_IDEN            = 12
	UCR1_ICD             = 10
	UCR1_RRDYEN          = 9
	UCR1_RXDMAEN         = 8
	UCR1_IREN            = 7
	UCR1_TXMPTYEN        = 6
	UCR1_RTSDEN          = 5
	UCR1_SNDBRK          = 4
	UCR1_TXDMAEN         = 3
	UCR1_ATDMAEN         = 2
	UCR1_DOZE            = 1
	UCR1_UARTEN          = 0

	UARTx_UCR2 uint32 = 0x0084
	UCR2_ESCI         = 15
	UCR2_IRTS         = 14
	UCR2_CTSC         = 13
	UCR2_CTS          = 12
	UCR2_ESCEN        = 11
	UCR2_RTEC         = 9
	UCR2_PREN         = 8
	UCR2_PROE         = 7
	UCR2_STPB         = 6
	UCR2_WS           = 5
	UCR2_RTSEN        = 4
	UCR2_ATEN         = 3
	UCR2_TXEN         = 2
	UCR2_RXEN         = 1
	UCR2_SRST         = 0

	UARTx_UCR3     uint32 = 0x0088
	UCR3_DPEC             = 14
	UCR3_DTREN            = 13
	UCR3_PARERREN         = 12
	UCR3_FRAERREN         = 11
	UCR3_DSR              = 10
	UCR3_DCD              = 9
	UCR3_RI               = 8
	UCR3_ADNIMP           = 7
	UCR3_RXDSEN           = 6
	UCR3_AIRINTEN         = 5
	UCR3_AWAKEN           = 4
	UCR3_DTRDEN           = 3
	UCR3_RXDMUXSEL        = 2
	UCR3_INVT             = 1
	UCR3_ACIEN            = 0

	UARTx_UCR4 uint32 = 0x008c
	UCR4_CTSTL        = 10

	UARTx_UFCR  uint32 = 0x0090
	UFCR_TXTL          = 10
	UFCR_RFDIV         = 7
	UFCR_DCEDTE        = 6
	UFCR_RXTL          = 0

	UARTx_USR2 uint32 = 0x0098
	USR2_RDR          = 0

	UARTx_UESC        = 0x009c
	UARTx_UTIM        = 0x00a0
	UARTx_UBIR        = 0x00a4
	UARTx_UBMR uint32 = 0x00a8

	UARTx_UTS   uint32 = 0x00b4
	UTS_TXEMPTY        = 6
)

type Uart struct {
	sync.Mutex

	urxd uint32
	utxd uint32
	ucr1 uint32
	ucr2 uint32
	ucr3 uint32
	ucr4 uint32
	ufcr uint32
	usr2 uint32
	uesc uint32
	utim uint32
	ubir uint32
	ubmr uint32
	uts  uint32
}

// UART2 instance
var UART2 = &Uart{}

func (hw *Uart) init(base uint32, baudrate uint32) {
	hw.urxd = base + UARTx_URXD
	hw.utxd = base + UARTx_UTXD
	hw.ucr1 = base + UARTx_UCR1
	hw.ucr2 = base + UARTx_UCR2
	hw.ucr3 = base + UARTx_UCR3
	hw.ucr4 = base + UARTx_UCR4
	hw.ufcr = base + UARTx_UFCR
	hw.usr2 = base + UARTx_USR2
	hw.uesc = base + UARTx_UESC
	hw.utim = base + UARTx_UTIM
	hw.ubir = base + UARTx_UBIR
	hw.ubmr = base + UARTx_UBMR
	hw.uts = base + UARTx_UTS

	hw.Init(baudrate)
}

func uartclk() uint32 {
	var freq uint32

	if reg.Get(CCM_CSCDR1, CSCDR1_UART_CLK_SEL, 0b1) == 1 {
		freq = OSC_FREQ
	} else {
		freq = VCO_FREQ
	}

	podf := reg.Get(CCM_CSCDR1, CSCDR1_CLK_PODF, 0b111111)

	return freq / (podf + 1)
}

func (hw *Uart) txEmpty() bool {
	return reg.Get(hw.uts, UTS_TXEMPTY, 1) == 0
}

func (hw *Uart) rxReady() bool {
	return reg.Get(hw.usr2, USR2_RDR, 1) == 1
}

func (hw *Uart) rxError() bool {
	return reg.Get(hw.urxd, URXD_PRERR, 0b11111) != 0
}

// Setup programs the UART for RS-232 mode with the requested baudrate,
// p2312, 45.13.1 Programming the UART in RS-232 mode, IMX6ULLRM.
func (hw *Uart) Init(baudrate uint32) {
	hw.Lock()

	// disable UART
	reg.Write(hw.ucr1, 0)
	reg.Write(hw.ucr2, 0)

	// wait for software reset deassertion
	reg.Wait(hw.ucr2, UCR2_SRST, 1, 1)

	var ucr3 uint32
	// Data Set Ready
	bits.Set(&ucr3, UCR3_DSR)
	// Data Carrier Detect
	bits.Set(&ucr3, UCR3_DCD)
	// Ring Indicator
	bits.Set(&ucr3, UCR3_RI)
	// Disable autobaud detection
	bits.Set(&ucr3, UCR3_ADNIMP)
	// UART is in MUXED mode
	bits.Set(&ucr3, UCR3_RXDMUXSEL)
	// set UCR3
	reg.Write(hw.ucr3, ucr3)

	// 32 characters in the RxFIFO (maximum)
	reg.SetN(hw.ucr4, UCR4_CTSTL, 0b111111, 32)
	// set escape character
	reg.Write(hw.uesc, ESC)
	// reset escape timer
	reg.Write(hw.utim, 0)

	var ufcr uint32
	// Divide input clock by 2
	bits.SetN(&ufcr, UFCR_RFDIV, 0b111, 0b100)
	// TxFIFO has 2 or fewer characters
	bits.SetN(&ufcr, UFCR_TXTL, 0b111111, 2)
	// RxFIFO has 1 character
	bits.SetN(&ufcr, UFCR_RXTL, 0b111111, 1)
	// set UFCR
	reg.Write(hw.ufcr, ufcr)

	// p2299, 45.5 Binary Rate Multiplier (BRM), IMX6ULLRM
	//
	//              ref_clk_freq
	// baudrate = -----------------
	//                   UBMR + 1
	//             16 * ----------
	//                   UBIR + 1
	//
	// ref_clk_freq = module_clock

	// match /6 static divider (p424, Figure 17-3. Clock Tree - Part 2, IMX6ULLRM)
	clk := uartclk() / 6
	// multiply to match UFCR_RFDIV divider value
	ubmr := clk / (2 * baudrate)
	// neutralize denominator
	reg.Write(hw.ubir, 15)
	// set UBMR
	reg.Write(hw.ubmr, ubmr)

	var ucr2 uint32
	// Ignore the RTS pin
	bits.Set(&ucr2, UCR2_IRTS)
	// 8-bit transmit and receive character length
	bits.Set(&ucr2, UCR2_WS)
	// Enable the transmitter
	bits.Set(&ucr2, UCR2_TXEN)
	// Enable the receiver
	bits.Set(&ucr2, UCR2_RXEN)
	// Software reset
	bits.Set(&ucr2, UCR2_SRST)
	// set UCR2
	reg.Write(hw.ucr2, ucr2)

	// Enable the UART
	reg.Set(hw.ucr1, UCR1_UARTEN)

	hw.Unlock()
}

// Write a single character to the selected serial port.
func (hw *Uart) Write(c byte) {
	// transmit data
	reg.Write(hw.utxd, uint32(c))

	for hw.txEmpty() {
		// wait for TX FIFO to be empty
	}
}

// Read a single character from the selected serial port.
func (hw *Uart) Read() (c byte, valid bool) {
	if !hw.rxReady() {
		return
	}

	if hw.rxError() {
		return
	}

	return byte(reg.Get(hw.urxd, URXD_RX_DATA, 0xff)), true
}
