# QOI - The “Quite OK Image” format for fast, lossless image compression - in Go

Fork of [xfmoulet/qoi](https://github.com/xfmoulet/qoi) with changes to allow buffer reuse. See `qoi.DecodeIntoBuffer()`. The version in this repository will also:
* omit the alpha channel if it is not present when decoding.
* omit the alpha channel if it is not present **or** not used (all alpha values =100%) when encoding.
* write un-premultiplied instead of premultiplied values. (in accordance with QOI specification)

See [qoi.h](https://github.com/phoboslab/qoi/blob/master/qoi.h) for format specification.

More info at https://qoiformat.org/ 

## Tests?

Feel free to add.
