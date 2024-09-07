//go:build windows

package game

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Source code in this file was copied from https://github.com/jeandeaual/go-locale
// The following license applies to the source code in this file:
//
// MIT License
//
// Copyright (c) 2020 Alexis Jeandeau
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// LocaleNameMaxLength is the maximum length of a locale name on Windows.
// See https://docs.microsoft.com/en-us/windows/win32/intl/locale-name-constants.
const LocaleNameMaxLength uint32 = 85

func splitLocale(locale string) (string, string) {
	// Remove the encoding, if present.
	formattedLocale, _, _ := strings.Cut(locale, ".")

	// Normalize by replacing the hyphens with underscores
	formattedLocale = strings.ReplaceAll(formattedLocale, "-", "_")

	// Split at the underscore.
	language, territory, _ := strings.Cut(formattedLocale, "_")
	return language, territory
}

func getWindowsLocaleFromProc(syscall string) (string, error) {
	dll, err := windows.LoadDLL("kernel32")
	if err != nil {
		return "", fmt.Errorf("could not find the kernel32 DLL: %v", err)
	}

	proc, err := dll.FindProc(syscall)
	if err != nil {
		return "", fmt.Errorf("could not find the %s proc in kernel32: %v", syscall, err)
	}

	buffer := make([]uint16, LocaleNameMaxLength)

	// See https://docs.microsoft.com/en-us/windows/win32/api/winnls/nf-winnls-getuserdefaultlocalename
	// and https://docs.microsoft.com/en-us/windows/win32/api/winnls/nf-winnls-getsystemdefaultlocalename
	// GetUserDefaultLocaleName and GetSystemDefaultLocaleName both take a buffer and a buffer size,
	// and return the length of the locale name (0 if not found).
	ret, _, err := proc.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(LocaleNameMaxLength))
	if ret == 0 {
		return "", fmt.Errorf("locale not found when calling %s: %v", syscall, err)
	}

	return windows.UTF16ToString(buffer), nil
}

func getWindowsLocale() (string, error) {
	var (
		locale string
		err    error
	)

	for _, proc := range [...]string{"GetUserDefaultLocaleName", "GetSystemDefaultLocaleName"} {
		locale, err = getWindowsLocaleFromProc(proc)
		if err == nil {
			return locale, err
		}
	}

	return locale, err
}

// GetLocale retrieves the IETF BCP 47 language tag set on the system.
func GetLocale() (string, error) {
	locale, err := getWindowsLocale()
	if err != nil {
		return "", fmt.Errorf("cannot determine locale: %v", err)
	}

	return locale, err
}
