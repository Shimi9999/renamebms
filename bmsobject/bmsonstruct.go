package bmsobject

type Bmson struct {
  Bmsoninfo BmsonInfo `json:"info"`
}

type BmsonInfo struct {
  Title string `json:"title"`
  Subtitle string `json:"subtitle"`
  Chartname string `json:"chart_name"`
  Artist string `json:"artist"`
  Genre string `json:"genre"`
  Level int `json:"level"`
  Modehint string `json:"mode_hint"`
}
