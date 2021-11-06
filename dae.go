package jank3d

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/kvartborg/vector"
)

type daeSource struct {
	ID          string `xml:"id,attr"`
	StringArray string `xml:"float_array"`
}

func (source daeSource) Parse() []float64 {
	split := strings.Split(strings.ReplaceAll(source.StringArray, "\n", " "), " ")
	data := []float64{}
	for _, v := range split {
		f, _ := strconv.ParseFloat(v, 64)
		data = append(data, f)
	}
	return data
}

type daeTrianglesInput struct {
	Source string `xml:"source,attr"`
	Offset int    `xml:"offset,attr"`
}

type daeGeometry struct {
	Name           string              `xml:"name,attr"`
	Sources        []daeSource         `xml:"mesh>source"`
	TriangleString string              `xml:"mesh>triangles>p"`
	TriangleInputs []daeTrianglesInput `xml:"mesh>triangles>input"`
}

type libraryGeometries struct {
	Geometries []daeGeometry `xml:"library_geometries>geometry"`
}

func parseDAESourceName(sourceName string) string {

	idPieces := strings.Split(sourceName, "-")

	for endingIndex := len(idPieces) - 1; endingIndex >= 0; endingIndex-- {

		ending := strings.ToLower(idPieces[endingIndex])

		if strings.Contains(ending, "color") {
			return "color"
		} else if strings.Contains(ending, "position") || strings.Contains(ending, "vert") {
			return "vertex"
		} else if strings.Contains(ending, "normal") {
			return "normal"
		} else if strings.Contains(ending, "map") {
			return "uv"
		}

	}

	return ""

}

// LoadMeshesFromDAEFile takes a filepath to a .dae model file, and returns a map consisting of *Mesh objects, keyed by their names.
// If the call couldn't complete for any reason, like due to a malformed DAE file, it will return an error.
func LoadMeshesFromDAEFile(path string) (map[string]*Mesh, error) {

	if fileData, err := os.ReadFile(path); err != nil {
		return nil, err
	} else {
		return LoadMeshesFromDAEData(fileData)
	}

}

// LoadMeshFromDAEFile takes a []byte consisting of the contents of a DAE file, and returns a map consisting of *Mesh objects, keyed by their names.
// If the call couldn't complete for any reason, like due to a malformed DAE file, it will return an error.
func LoadMeshesFromDAEData(data []byte) (map[string]*Mesh, error) {

	meshes := map[string]*Mesh{}

	daeGeo := &libraryGeometries{}

	err := xml.Unmarshal(data, daeGeo)

	toYUp := Rotate(1, 0, 0, math.Pi/2)

	if err == nil {

		for _, geo := range daeGeo.Geometries {

			sourceData := map[string][]float64{}

			for _, source := range geo.Sources {

				if parsedName := parseDAESourceName(source.ID); parsedName != "" {
					sourceData[parsedName] = source.Parse()
				} else {
					fmt.Println("Unknown geometry data, ", source.ID)
				}

			}

			// triangleOrder := append([]daeTrianglesInput{}, geo.TriangleInputs...)
			// sort.Slice(triangleOrder, func(i, j int) bool { return triangleOrder[i].Offset < triangleOrder[j].Offset })
			triangleOrder := []string{}
			for _, t := range geo.TriangleInputs {
				triangleOrder = append(triangleOrder, parseDAESourceName(t.Source))
			}
			sort.Slice(triangleOrder, func(i, j int) bool { return geo.TriangleInputs[i].Offset < geo.TriangleInputs[j].Offset })

			triangleIndices := []int64{}

			for _, t := range strings.Split(strings.TrimSpace(geo.TriangleString), " ") {

				var ti int64
				ti, err = strconv.ParseInt(t, 0, 32)
				if err != nil {
					return nil, err
				}

				triangleIndices = append(triangleIndices, ti)

			}

			verts := []*Vertex{}
			normals := map[*Vertex]vector.Vector{}

			x, y, z := 0.0, 0.0, 0.0
			u, v := 0.0, 0.0
			r, g, b, a := float32(1.0), float32(1.0), float32(1.0), float32(1.0)
			nx, ny, nz := 0.0, 0.0, 0.0

			for i := 0; i < len(triangleIndices); i++ {

				triIndex := triangleIndices[i]

				tv := int(triIndex)

				switch triangleOrder[i%len(triangleOrder)] {

				case "vertex":
					x = sourceData["vertex"][tv*3]
					y = sourceData["vertex"][(tv*3)+1]
					z = sourceData["vertex"][(tv*3)+2]

				case "uv":
					u = sourceData["uv"][tv*2]
					v = sourceData["uv"][(tv*2)+1]

				case "color":
					r = float32(sourceData["color"][tv*4])
					g = float32(sourceData["color"][(tv*4)+1])
					b = float32(sourceData["color"][(tv*4)+2])
					a = float32(sourceData["color"][(tv*4)+3])

				case "normal":

					nx = sourceData["normal"][tv*3]
					ny = sourceData["normal"][(tv*3)+1]
					nz = sourceData["normal"][(tv*3)+2]

				}

				if i%len(triangleOrder) == len(triangleOrder)-1 {

					vert := NewVertex(x, y, z, u, v)

					vert.Color.R = r
					vert.Color.G = g
					vert.Color.B = b
					vert.Color.A = a

					verts = append(verts, vert)

					normals[vert] = vector.Vector{nx, ny, nz}

				}

			}

			mesh := NewMesh(geo.Name, verts...)

			if len(normals) > 0 {

				for _, tri := range mesh.Triangles {

					normal := vector.Vector{0, 0, 0}
					for _, vert := range tri.Vertices {
						normal = normal.Add(normals[vert])
					}
					normal = normal.Scale(1.0 / 3.0).Unit()

					normal = toYUp.MultVec(normal)
					normal[2] *= -1
					normal[1] *= -1
					tri.Normal = normal

				}

			}

			meshes[geo.Name] = mesh

		}

	}

	return meshes, err

}