package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/samuelyuan/Civ5MapViewer/internal/fileio"
	"github.com/samuelyuan/Civ5MapViewer/internal/api"
	"github.com/samuelyuan/Civ5MapViewer/internal/mapdraw"
)

type editor struct {
	drawSurface             *interactiveRaster
	status                  *widget.Label
	cache                   *image.RGBA
	cacheWidth, cacheHeight int
	hexTileProperties       *widget.Label

	uri  string
	img  *image.RGBA
	mapMode string
	mapData *fileio.Civ5MapData
	mapHeight int
	mapWidth int
	currentHexProperties string
	zoom int

	win        fyne.Window
	recentMenu *fyne.Menu
}

func (e *editor) PixelColor(x, y int) color.Color {
	return e.img.At(x, y)
}

func colorToBytes(col color.Color) []uint8 {
	r, g, b, a := col.RGBA()
	return []uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func (e *editor) Clicked(x, y int, edit api.Editor) {
	edit.SetHexCoordinates(x, y)
}

func (e *editor) SetMapMode(mapMode string) {
	e.mapMode = mapMode
	if e.uri == "" {
		return
	}

	u, _ := storage.ParseURI(e.uri)
	read, err := storage.Reader(u)
	if err != nil {
		fyne.LogError("Unable to open file \""+e.uri+"\"", err)
		return
	}
	fmt.Println("Reload map with new mode:", e.mapMode)
	e.LoadFile(read, e.mapMode)
}

func (e *editor) SetHexCoordinates(x int, y int) {
	 hexX, hexY := mapdraw.GetHexCoordinates(x, y, e.mapHeight, e.mapWidth)
	 e.currentHexProperties = ""
	 e.currentHexProperties += fmt.Sprintf("Plot (%d, %d)\n", hexX, hexY)

	 tile := e.mapData.MapTileImprovements[hexY][hexX]
	 physicalTile := e.mapData.MapTiles[hexY][hexX]

	 e.currentHexProperties += fmt.Sprintf("City: %s\n", tile.CityName)
	 e.currentHexProperties += fmt.Sprintf("Owner: Player %d\n", tile.Owner)

	 terrain := "Unknown"
	 if physicalTile.TerrainType >= 0 && int(physicalTile.TerrainType) < len(e.mapData.TerrainList) {
		 terrain = e.mapData.TerrainList[physicalTile.TerrainType]
	 }
	 e.currentHexProperties += fmt.Sprintf("Terrain: %s\n", terrain)

	 elevation := "Unknown"
	 if physicalTile.Elevation == 0 {
		 elevation = "Flat Terrain"
	 } else if physicalTile.Elevation == 1 {
		 elevation = "Hills"
	 } else if physicalTile.Elevation == 2 {
		 elevation = "Mountains"
	 }
	 e.currentHexProperties += fmt.Sprintf("Elevation: %s\n", elevation)

	 resource := "Unknown"
	 if physicalTile.ResourceType >= 0 && int(physicalTile.ResourceType) < len(e.mapData.ResourceList) {
		 resource = e.mapData.ResourceList[physicalTile.ResourceType]
	 } else if physicalTile.ResourceType == 255 {
		 resource = "No Resource"
	 }
	 e.currentHexProperties += fmt.Sprintf("Resource: %s\n", resource)

	 improvement := "Unknown"
	 if tile.Improvement >= 0 && int(tile.Improvement) < len(e.mapData.TileImprovementList) {
		 improvement = e.mapData.TileImprovementList[tile.Improvement]
	 } else if tile.Improvement == 255 {
		 improvement = "No Improvement"
	 }
	 e.currentHexProperties += fmt.Sprintf("Tile Improvement:%s\n", improvement)

	 routeType := "Unknown"
	 if tile.RouteType == 0 {
		 routeType = "Road"
	 } else if tile.RouteType == 1 {
		 routeType = "Railroad"
	 } else if tile.RouteType == 255 {
		 routeType = "No Route"
	 }
	 e.currentHexProperties += fmt.Sprintf("Route: %s\n", routeType)

	 e.hexTileProperties.SetText(e.currentHexProperties)
}

func (e *editor) buildUI() fyne.CanvasObject {
	return container.NewScroll(e.drawSurface)
}

func (e *editor) setZoom(zoom int) {
	e.zoom = zoom
	e.updateSizes()
	e.drawSurface.Refresh()
}

func (e *editor) draw(w, h int) image.Image {
	if e.cacheWidth == 0 || e.cacheHeight == 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	if w > e.cacheWidth || h > e.cacheHeight {
		bigger := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(bigger, e.cache.Bounds(), e.cache, image.Point{}, draw.Over)
		return bigger
	}

	return e.cache
}

func (e *editor) updateSizes() {
	if e.img == nil {
		return
	}
	e.cacheWidth = e.img.Bounds().Dx() * e.zoom
	e.cacheHeight = e.img.Bounds().Dy() * e.zoom

	c := fyne.CurrentApp().Driver().CanvasForObject(e.status)
	scale := float32(1.0)
	if c != nil {
		scale = c.Scale()
	}
	e.drawSurface.SetMinSize(fyne.NewSize(
		float32(e.cacheWidth)/scale,
		float32(e.cacheHeight)/scale))

	e.renderCache()
}

func (e *editor) pixAt(x, y int) []uint8 {
	ix := x / e.zoom
	iy := y / e.zoom

	if ix >= e.img.Bounds().Dx() || iy >= e.img.Bounds().Dy() {
		return []uint8{0, 0, 0, 0}
	}

	return colorToBytes(e.img.At(ix, iy))
}

func (e *editor) renderCache() {
	e.cache = image.NewRGBA(image.Rect(0, 0, e.cacheWidth, e.cacheHeight))
	for y := 0; y < e.cacheHeight; y++ {
		for x := 0; x < e.cacheWidth; x++ {
			i := (y*e.cacheWidth + x) * 4
			col := e.pixAt(x, y)
			e.cache.Pix[i] = col[0]
			e.cache.Pix[i+1] = col[1]
			e.cache.Pix[i+2] = col[2]
			e.cache.Pix[i+3] = col[3]
		}
	}

	e.drawSurface.Refresh()
}

func fixEncoding(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}

	newImg := image.NewRGBA(img.Bounds())
	draw.Draw(newImg, newImg.Bounds(), img, img.Bounds().Min, draw.Over)
	return newImg
}

func (e *editor) LoadFile(read fyne.URIReadCloser, mapMode string) {
	defer read.Close()

	inputFilename := read.URI().String()[7:]
	fmt.Println("Input filename: ", inputFilename)
	inputFileExtension := filepath.Ext(inputFilename)

	var mapData *fileio.Civ5MapData
	var err error
	if inputFileExtension == ".json" {
		fmt.Println("Importing map file from json")
		mapData = fileio.ImportCiv5MapFileFromJson(inputFilename)
		mapdraw.OverrideColorMap(mapData.CivColorOverrides)
	} else {
		fmt.Println("Reading civ5map file")
		mapData, err = fileio.ReadCiv5MapFile(inputFilename)
		if err != nil {
			fyne.LogError("Failed to read input file: ", err)
			e.status.SetText(err.Error())
			return
		}
	}


	var img image.Image
	if e.mapMode == "physical" {
		img = mapdraw.DrawPhysicalMap(mapData)
	} else {
		img = mapdraw.DrawPoliticalMap(mapData)
	}

	e.addRecent(read.URI())
	e.uri = read.URI().String()
	e.img = fixEncoding(img)
	e.mapData = mapData
	e.mapHeight = len(mapData.MapTiles)
	e.mapWidth = len(mapData.MapTiles[0])
	e.status.SetText(fmt.Sprintf("File: %s | Width: %d | Height: %d | Map Rows: %d | Map Cols: %d",
		filepath.Base(read.URI().String()), e.img.Bounds().Dx(), e.img.Bounds().Dy(), e.mapHeight, e.mapWidth))
	e.updateSizes()
}

func (e *editor) Reload() {
	if e.uri == "" {
		return
	}

	u, _ := storage.ParseURI(e.uri)
	read, err := storage.Reader(u)
	if err != nil {
		fyne.LogError("Unable to open file \""+e.uri+"\"", err)
		return
	}
	e.LoadFile(read, e.mapMode)
}

func (e *editor) WriteJson(write fyne.URIWriteCloser) {
	if e.uri == "" {
		return
	}
	defer write.Close()
	if write.URI().Extension() == ".json" {
		fmt.Println("Exporting map to", write.URI().Path())
		fileio.ExportCiv5MapFile(e.mapData, write.URI().Path())
	}
}

func (e *editor) Save() {
	if e.uri == "" {
		return
	}

	uri, _ := storage.ParseURI(e.uri)
	if !e.isSupported(uri.Extension()) {
		fyne.LogError("Save only supports PNG", nil)
		return
	}
	write, err := storage.Writer(uri)
	if err != nil {
		fyne.LogError("Error opening file to replace", err)
		return
	}

	e.saveToWriter(write)
}

func (e *editor) saveToWriter(write fyne.URIWriteCloser) {
	defer write.Close()
	if e.isPNG(write.URI().Extension()) {
		err := png.Encode(write, e.img)

		if err != nil {
			fyne.LogError("Could not encode image", err)
		}
	}
}

func (e *editor) SaveAs(writer fyne.URIWriteCloser) {
	e.saveToWriter(writer)
}

func (e *editor) isSupported(path string) bool {
	return e.isPNG(path)
}

func (e *editor) isPNG(path string) bool {
	return strings.LastIndex(strings.ToLower(path), "png") == len(path)-3
}

// NewEditor creates a new pixel editor that is ready to have a file loaded
func NewEditor() api.Editor {
	hexTileProperties := widget.NewLabel("Tile Properties")
	edit := &editor{zoom: 1, hexTileProperties: hexTileProperties, status: newStatusBar(), mapMode: "political"}
	edit.drawSurface = newInteractiveRaster(edit)

	return edit
}
