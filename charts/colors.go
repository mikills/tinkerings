package main

// color utility for chart.js configurations
//
// usage examples:
//
// bar chart with custom colors:
//   {
//     "points": [...],
//     "backgroundColors": ["rgba(255, 99, 132, 0.8)", "rgba(54, 162, 235, 0.8)"],
//     "borderColors": ["rgba(255, 99, 132, 1)", "rgba(54, 162, 235, 1)"]
//   }
//
// line/area/scatter chart with single color:
//   {
//     "points": [...],
//     "backgroundColor": "rgba(75, 192, 192, 0.2)",
//     "borderColor": "rgba(75, 192, 192, 1)"
//   }
//
// if colors are omitted, charts will use the default palette automatically

// default color palette with vibrant, accessible colors
var defaultColorPalette = []string{
	"rgba(54, 162, 235, 0.8)",  // blue
	"rgba(255, 99, 132, 0.8)",  // red
	"rgba(75, 192, 192, 0.8)",  // teal
	"rgba(255, 206, 86, 0.8)",  // yellow
	"rgba(153, 102, 255, 0.8)", // purple
	"rgba(255, 159, 64, 0.8)",  // orange
	"rgba(46, 204, 113, 0.8)",  // green
	"rgba(231, 76, 60, 0.8)",   // dark red
	"rgba(52, 152, 219, 0.8)",  // medium blue
	"rgba(241, 196, 15, 0.8)",  // gold
	"rgba(155, 89, 182, 0.8)",  // violet
	"rgba(26, 188, 156, 0.8)",  // turquoise
}

// border colors with full opacity
var defaultBorderColors = []string{
	"rgba(54, 162, 235, 1)",
	"rgba(255, 99, 132, 1)",
	"rgba(75, 192, 192, 1)",
	"rgba(255, 206, 86, 1)",
	"rgba(153, 102, 255, 1)",
	"rgba(255, 159, 64, 1)",
	"rgba(46, 204, 113, 1)",
	"rgba(231, 76, 60, 1)",
	"rgba(52, 152, 219, 1)",
	"rgba(241, 196, 15, 1)",
	"rgba(155, 89, 182, 1)",
	"rgba(26, 188, 156, 1)",
}

// getColors returns colors for the specified count
// if userColors provided and sufficient, use those; otherwise use default palette
func getColors(userColors []string, count int) []string {
	if len(userColors) >= count {
		return userColors[:count]
	}

	colors := make([]string, count)
	for i := 0; i < count; i++ {
		colors[i] = defaultColorPalette[i%len(defaultColorPalette)]
	}
	return colors
}

// getBorderColors returns border colors for the specified count
func getBorderColors(userColors []string, count int) []string {
	if len(userColors) >= count {
		return userColors[:count]
	}

	colors := make([]string, count)
	for i := 0; i < count; i++ {
		colors[i] = defaultBorderColors[i%len(defaultBorderColors)]
	}
	return colors
}

// getSingleColor returns a single color from user colors or default palette
func getSingleColor(userColors []string, index int) string {
	if len(userColors) > index {
		return userColors[index]
	}
	return defaultColorPalette[index%len(defaultColorPalette)]
}

// getSingleBorderColor returns a single border color from user colors or default palette
func getSingleBorderColor(userColors []string, index int) string {
	if len(userColors) > index {
		return userColors[index]
	}
	return defaultBorderColors[index%len(defaultBorderColors)]
}
