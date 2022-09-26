# QOI - The “Quite OK Image” format for fast, lossless image compression - in Go

Fork of github.com/xfmoulet/qoi where `image.Image` has been replaced with `[]byte` to allow the caller to reuse memory regions for better speeds when loading many images in succession (in my case, specifically with the intent to serve the data to OpenGL). The version in this repository will also omit the alpha channel if it is not present or not used (non-255 values) when decoding QOI image data as well as write the correct channel count when encoding.

See [qoi.h](https://github.com/phoboslab/qoi/blob/master/qoi.h) for the documentation.

More info at https://qoiformat.org/ 

## Tests?

Feel free to add.
