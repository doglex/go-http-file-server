package serverHandler

import (
	"../serverError"
	"../util"
	"os"
	"path"
	"strings"
)

func (h *handler) visitFs(
	initFsPath, rawRequestPath, relPath string,
	callback func(*os.File, os.FileInfo, string),
) {
	aliasedFsPath, hasAlias := h.aliases[rawRequestPath]

	var fsPath string
	if hasAlias {
		fsPath = aliasedFsPath
	} else {
		fsPath = initFsPath
	}

	f, err := os.Open(fsPath)
	serverError.LogError(err)

	var fInfo os.FileInfo
	if err != nil {
		if os.IsExist(err) {
			return
		}
		fInfo = newFakeFileInfo(path.Base(fsPath), true)
	} else {
		fInfo, err = f.Stat()
		if serverError.LogError(err) {
			return
		}
	}

	if len(relPath) > 0 {
		callback(f, fInfo, relPath)
	}

	if fInfo.IsDir() {
		childAliases := map[string]string{}
		for aliasUrlPath, aliasFsPath := range h.aliases {
			if path.Dir(aliasUrlPath) == rawRequestPath {
				childAliases[aliasUrlPath] = aliasFsPath
				continue
			}

			var rawRequestPathDir string
			if rawRequestPath == "/" {
				rawRequestPathDir = rawRequestPath
			} else {
				rawRequestPathDir = rawRequestPath + "/"
			}
			if strings.HasPrefix(aliasUrlPath, rawRequestPathDir) {
				succPath := aliasUrlPath[len(rawRequestPath):]
				if succPath[0] == '/' {
					succPath = succPath[1:]
				}
				childName := succPath[:strings.Index(succPath, "/")]
				childUrlPath := util.CleanUrlPath(rawRequestPath + "/" + childName)
				childFsPath := fsPath + "/" + childName
				childAliases[childUrlPath] = childFsPath
				continue
			}
		}

		var childInfos []os.FileInfo
		if f != nil {
			childInfos, err = f.Readdir(0)
			if serverError.LogError(err) {
				return
			}
		}

		for _, childInfo := range childInfos {
			childPath := "/" + childInfo.Name()
			childFsPath := fsPath + childPath
			childRawRequestPath := util.CleanUrlPath(rawRequestPath + childPath)
			childRelPath := relPath + childPath

			if childAliasedFsPath, hasChildAlias := childAliases[childRawRequestPath]; hasChildAlias {
				h.visitFs(childAliasedFsPath, childRawRequestPath, childRelPath, callback)
				delete(childAliases, childRawRequestPath)
			} else {
				h.visitFs(childFsPath, childRawRequestPath, childRelPath, callback)
			}
		}

		for childRawRequestPath, childAliasedFsPath := range childAliases {
			childRelPath := relPath + "/" + path.Base(childRawRequestPath)
			h.visitFs(childAliasedFsPath, childRawRequestPath, childRelPath, callback)
		}

	}
}