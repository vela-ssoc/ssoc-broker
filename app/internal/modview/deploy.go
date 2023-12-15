package modview

import "net/url"

type Deploy struct {
	DownloadURL *url.URL
}

func (d Deploy) WindowsDownloadURL() string {
	downloadURL := d.DownloadURL
	if downloadURL == nil {
		return ""
	}

	strRunes := []rune(downloadURL.String())
	ret := make([]rune, 0, len(strRunes))
	for _, r := range strRunes {
		if r == '%' {
			ret = append(ret, '%')
		}
		ret = append(ret, r)
	}

	return string(ret)
}
