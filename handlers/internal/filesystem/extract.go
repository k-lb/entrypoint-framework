/*
 *  Copyright (c) 2023 Samsung Electronics Co., Ltd All Rights Reserved
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License
 */

package filesystem

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Extract extracts all files from a tarball to a toDir directory. If any errors occurs or anything from the tarball is
// not a regular file, directory, hardlink or symlink then an error is returned.
func (real) Extract(tarball, toDir string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return fmt.Errorf("could not open %s. Reason: %w", tarball, err)
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("could not extract a file %s. Reason: %w", tarball, err)
		}
		path := filepath.Join(toDir, header.Name)
		info := header.FileInfo()

		switch header.Typeflag {
		case tar.TypeReg:
			file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
			if err != nil {
				return fmt.Errorf("could not open a file %s from %s. Reason: %w", path, tarball, err)
			}
			defer file.Close()

			_, err = io.Copy(file, tarReader)
			if err != nil {
				return fmt.Errorf("could not copy a file %s from %s. Reason: %w", path, tarball, err)
			}
		case tar.TypeDir:
			if err := os.MkdirAll(path, info.Mode()); err != nil {
				return fmt.Errorf("could not create a directory %s from %s. Reason: %w", path, tarball, err)
			}
		case tar.TypeLink:
			linkPath := filepath.Join(toDir, header.Linkname)
			if path != linkPath {
				if err := os.Link(linkPath, path); err != nil {
					return fmt.Errorf("could not create a hardlink from %s to %s from %s. Reason: %w", linkPath, path, tarball, err)
				}
			}
		case tar.TypeSymlink:
			linkPath := filepath.Join(toDir, header.Linkname[len(filepath.Dir(tarball)):])
			if err := os.Symlink(linkPath, path); err != nil {
				return fmt.Errorf("could not create a hardlink from %s to %s from %s. Reason: %w", linkPath, path, tarball, err)
			}
		default:
			return fmt.Errorf("%s from %s is not a directory, regular file, hardlink or symlink", header.Name, tarball)
		}
	}
	return nil
}
