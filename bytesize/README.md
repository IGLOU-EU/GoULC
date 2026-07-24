# 📏 ByteSize Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/bytesize.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/bytesize)

A Go package for handling byte sizes using IEC binary units (powers of 1024). It provides parsing, formatting, and arithmetic operations for byte sizes from Bytes up to Pebibytes.

## 🎯 Features

- **🔌 Interfacing**:
  - JSON Marshaler/Unmarshaler for IEC string representation
  - JSON numbers are accepted as byte counts, with the same range checks and fraction handling as strings
  - Text Marshaler/Unmarshaler, enabling JSON map keys and `flag.TextVar`
  - Stringer for human-readable output
  - `IsZero()` support, so the `omitzero` JSON tag option (Go 1.24+) omits zero sizes

- **💾 Dual Representation**:
  - Maintains both truncated integer (int64) and exact floating-point (float64) values
  - Provides canonical string representation (e.g., "42.5MiB")
  - Value accessors:
    - `Bytes()` returns truncated int64 (e.g., "42.42MiB" → 44480593)
    - `Exact()` returns float64 (e.g., "42.42MiB" → 44480593.92)
    - `String()` returns canonical string (e.g., "42.42MiB")

- **📊 Size Support**:
  - Range limited to what int64 can represent, ~8 EiB in both directions
  - Supports negative values and floating-point components
  - Truncates toward zero without rounding to prevent overflows
  - Integer overflow detection on both bounds
  - Non-finite values (NaN, ±Inf) are rejected at parsing, `ToString` formats them as-is ("+Inf", "-Inf", "NaN")

- **🔤 String Operations**:
  - Parse size strings in format "NUMBER[OPTIONNAL UNIT]" (e.g., "42", "42.5MiB", "1.2GiB")
  - Automatic unit selection for human-readable output
  - Converts short unit forms to IEC standard (e.g., "M" to "MiB")
  - Rounds to 2 decimal places (except for bytes)

- **🧮 Arithmetic Operations**:
  - Add sizes together with overflow protection
  - Work with raw byte counts or formatted strings

## 📐 Understanding Units

This package uses IEC binary units (powers of 1024) rather than SI decimal units (powers of 1000):

### 🖥️ IEC Binary Units (This Package)
- Uses powers of 1024 (2¹⁰)
- Clear "binary" notation with "i" (KiB, MiB, GiB)
- Matches how computers actually store data (binary notation)
```
1 KiB = 1024 bytes
1 MiB = 1024 KiB = 1,048,576 bytes
1 GiB = 1024 MiB = 1,073,741,824 bytes
```

### 📊 SI Decimal Units
- Uses powers of 1000
- Traditional notation (KB, MB, GB)
- Common in marketing and data transmission
```
1 KB = 1000 bytes
1 MB = 1000 KB = 1,000,000 bytes
1 GB = 1000 MB = 1,000,000,000 bytes
```

> 💡 **Example**: A "500 GB" hard drive using SI units (500,000,000,000 bytes) shows as "465.7 GiB" in most operating systems, which use IEC binary units internally.

## ⚠️ Important Notes

- Partial bytes not supported (e.g., "1.5 Bytes") - would require arbitrary byte width
- Failures are reported through exported sentinel error values, match them with `errors.Is`:
  - `ErrEmptyString`: empty string input
  - `ErrNoValue`: no numeric value found
  - `ErrInvalidIEC`: invalid IEC unit symbol
  - `ErrIntegerOverflow`: value not representable as an int64
  - `ErrJSONInvalidType`: invalid JSON input type

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
