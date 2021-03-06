// USB armory Mk II support for tamago/arm
// https://github.com/f-secure-foundry/tamago
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build !linkprintk

package usbarmory

import (
	_ "unsafe"

	"github.com/f-secure-foundry/tamago/soc/imx6"
)

// On the USB armory Mk II the serial console is UART2, therefore standard
// output is redirected there.
//
// The console is exposed through the USB Type-C receptacle and available only
// in debug accessory mode (see EnableDebugAccessory()).

//go:linkname printk runtime.printk
func printk(c byte) {
	imx6.UART2.Tx(c)
}
