package channelshift

import (
	"fmt"
	"image"
	"math"
	"math/rand"

	"github.com/accrazed/glitch-art/src/lib"
)

type NewOpt func(*ChannelShift) *ChannelShift

type ChannelShift struct {
	translate Translate
	image     *image.RGBA64
	seed      int64
	rand      *rand.Rand
	direction lib.Direction
	offsetVol int
	chunkVol  int
	chunk     int
	animate   int
}

type Translate struct {
	r image.Point
	g image.Point
	b image.Point
	a image.Point
}

func Must(ps *ChannelShift, err error) *ChannelShift {
	if err != nil {
		panic(err)
	}
	return ps
}

func New(path string, opts ...NewOpt) (*ChannelShift, error) {
	img, err := lib.NewImage(path)
	if err != nil {
		return nil, err
	}

	cs := &ChannelShift{
		image: img,
	}
	for _, opt := range opts {
		cs = opt(cs)
	}

	if cs.rand == nil {
		cs.rand = rand.New(rand.NewSource(0))
	}

	if cs.chunkVol > cs.chunk {
		cs.chunkVol = cs.chunk
	}

	if cs.chunk == 0 {
		if cs.image.Rect.Dx() > cs.image.Rect.Dy() {
			cs.chunk = cs.image.Rect.Dx()
		}
		cs.chunk = cs.image.Rect.Dy()
	}

	return cs, nil
}

func WithChunks(dist int) NewOpt {
	return func(cs *ChannelShift) *ChannelShift {
		cs.chunk = dist
		return cs
	}
}

func WithSeed(seed int64) NewOpt {
	return func(cs *ChannelShift) *ChannelShift {
		cs.seed = seed
		cs.rand = rand.New(rand.NewSource(seed))
		return cs
	}
}

func WithDirection(direction lib.Direction) NewOpt {
	return func(cs *ChannelShift) *ChannelShift {
		cs.direction = direction
		return cs
	}
}

func WithOffsetVolatility(offsetVol int) NewOpt {
	return func(cs *ChannelShift) *ChannelShift {
		cs.offsetVol = offsetVol
		return cs
	}
}

func WithChunkVolatility(chunkVol int) NewOpt {
	return func(cs *ChannelShift) *ChannelShift {
		cs.chunkVol = chunkVol
		return cs
	}
}

func WithAnimate(animate int) NewOpt {
	return func(cs *ChannelShift) *ChannelShift {
		cs.animate = animate
		return cs
	}
}

func RedShift(x, y int) NewOpt {
	return func(cs *ChannelShift) *ChannelShift {
		cs.translate.r.X = x
		cs.translate.r.Y = y
		return cs
	}
}

func GreenShift(x, y int) NewOpt {
	return func(cs *ChannelShift) *ChannelShift {
		cs.translate.g.X = x
		cs.translate.g.Y = y
		return cs
	}
}

func BlueShift(x, y int) NewOpt {
	return func(cs *ChannelShift) *ChannelShift {
		cs.translate.b.X = x
		cs.translate.b.Y = y
		return cs
	}
}

func AlphaShift(x, y int) NewOpt {
	return func(cs *ChannelShift) *ChannelShift {
		cs.translate.a.X = x
		cs.translate.a.Y = y
		return cs
	}
}

func (cs *ChannelShift) Shift() image.Image {
	numSlices, numPos := cs.image.Rect.Dx(), cs.image.Rect.Dy()
	if cs.direction == lib.Horizontal {
		numSlices, numPos = numPos, numSlices
	}

	outImg := lib.CopyImage(cs.image)
	offset := 0

	ch := make(chan bool)

	// Iterate through slices perpendicular to chunking directions
	for curSlice := 0; curSlice < numSlices; {
		chunkSize := cs.chunk
		if cs.chunkVol > 0 {
			chunkSize = cs.chunk + cs.rand.Intn(cs.chunkVol*2) - cs.chunkVol
			offset = (cs.rand.Int() % (cs.offsetVol * 2)) - cs.offsetVol
		}

		// Shift each slice
		var cur int
		for cur = 0; cur < chunkSize && cur+curSlice < numSlices; cur++ {
			go func(slice, offset int) {
				for pos := 0; pos < numPos; pos++ {
					sl, ps := numSlices, numPos
					if cs.direction == lib.Horizontal {
						sl, ps = ps, sl
						slice, pos = pos, slice
					}

					old := lib.RGBA64toPix(slice, pos, cs.image.Stride)

					rX, rY := (slice+cs.translate.r.X+offset)%sl,
						(pos+cs.translate.r.Y+offset)%ps
					gX, gY := (slice+cs.translate.g.X+offset)%sl,
						(pos+cs.translate.g.Y+offset)%ps
					bX, bY := (slice+cs.translate.b.X+offset)%sl,
						(pos+cs.translate.b.Y+offset)%ps
					aX, aY := (slice+cs.translate.a.X+offset)%sl,
						(pos+cs.translate.a.Y+offset)%ps

					newR := int(math.Abs(float64(lib.RGBA64toPix(rX, rY, cs.image.Stride))))
					newG := int(math.Abs(float64(lib.RGBA64toPix(gX, gY, cs.image.Stride))))
					newB := int(math.Abs(float64(lib.RGBA64toPix(bX, bY, cs.image.Stride))))
					newA := int(math.Abs(float64(lib.RGBA64toPix(aX, aY, cs.image.Stride))))

					// Red
					outImg.Pix[old+0] = cs.image.Pix[newR+0]
					outImg.Pix[old+1] = cs.image.Pix[newR+1]
					// Green
					outImg.Pix[old+2] = cs.image.Pix[newG+2]
					outImg.Pix[old+3] = cs.image.Pix[newG+3]
					// Blue
					outImg.Pix[old+4] = cs.image.Pix[newB+4]
					outImg.Pix[old+5] = cs.image.Pix[newB+5]
					// Alpha
					outImg.Pix[old+6] = cs.image.Pix[newA+6]
					outImg.Pix[old+7] = cs.image.Pix[newA+7]

					if cs.direction == lib.Horizontal {
						slice, pos = pos, slice
					}
				}
				ch <- true
			}(curSlice+cur, offset)
		}
		curSlice += cur
	}
	for i := 0; i < numSlices; i++ {
		<-ch
	}

	return outImg
}

func (cs *ChannelShift) ShiftIterate() []image.Image {
	res := make([]image.Image, 0)

	baseTr := cs.translate

	for i := 0; i < cs.animate; i++ {
		fmt.Printf("Generating channelshift frame %v...\n", i+1)
		res = append(res, cs.Shift())

		if cs.animate != 1 && cs.offsetVol != 0 {
			cs.translate.r.X = baseTr.r.X + rand.Int()%(cs.offsetVol*2) - cs.offsetVol
			cs.translate.r.Y = baseTr.r.Y + rand.Int()%(cs.offsetVol*2) - cs.offsetVol
			cs.translate.g.X = baseTr.g.X + rand.Int()%(cs.offsetVol*2) - cs.offsetVol
			cs.translate.g.Y = baseTr.g.Y + rand.Int()%(cs.offsetVol*2) - cs.offsetVol
			cs.translate.b.X = baseTr.b.X + rand.Int()%(cs.offsetVol*2) - cs.offsetVol
			cs.translate.b.Y = baseTr.b.Y + rand.Int()%(cs.offsetVol*2) - cs.offsetVol
			cs.translate.a.X = baseTr.a.X + rand.Int()%(cs.offsetVol*2) - cs.offsetVol
			cs.translate.a.Y = baseTr.a.Y + rand.Int()%(cs.offsetVol*2) - cs.offsetVol
		}
	}

	return res
}
