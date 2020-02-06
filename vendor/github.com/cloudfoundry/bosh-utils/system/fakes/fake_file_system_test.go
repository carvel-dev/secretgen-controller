package fakes_test

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshsys "github.com/cloudfoundry/bosh-utils/system"
	. "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("FakeFileSystem", func() {
	var (
		fs *FakeFileSystem
	)

	BeforeEach(func() {
		fs = NewFakeFileSystem()
	})

	Describe("MkdirAll", func() {
		It("creates a directory", func() {
			err := fs.MkdirAll("/potato", 0750)
			Expect(err).ToNot(HaveOccurred())

			fi, err := fs.Stat("/potato")
			Expect(err).ToNot(HaveOccurred())

			Expect(fi.IsDir()).To(BeTrue())
			Expect(fi.Mode()).To(Equal(os.FileMode(0750)))
		})

		It("works with Windows style pathing", func() {
			err := fs.MkdirAll("C:/potato", 0750)
			Expect(err).ToNot(HaveOccurred())

			fi, err := fs.Stat("C:/potato")
			Expect(err).ToNot(HaveOccurred())

			Expect(fi.IsDir()).To(BeTrue())
			Expect(fi.Mode()).To(Equal(os.FileMode(0750)))
		})

		It("can create the root directory", func() {
			err := fs.MkdirAll("/", 0750)
			Expect(err).ToNot(HaveOccurred())

			fi, err := fs.Stat("/")
			Expect(err).ToNot(HaveOccurred())

			Expect(fi.IsDir()).To(BeTrue())
			Expect(fi.Mode()).To(Equal(os.FileMode(0750)))
		})

		It("creates all directories it needs to", func() {
			segments := []string{"C:/potato", "with", "a", "subdirectory"}
			err := fs.MkdirAll(filepath.Join(segments...), 0750)
			Expect(err).ToNot(HaveOccurred())

			var path string
			for _, segment := range segments {
				path = filepath.Join(path, segment)

				fi, err := fs.Stat(path)
				Expect(err).ToNot(HaveOccurred())
				Expect(fi.IsDir()).To(BeTrue())
				Expect(fi.Mode()).To(Equal(os.FileMode(0750)))
			}
		})

		It("does not overwrite existing directories", func() {
			err := fs.MkdirAll("/potato", 0700)
			Expect(err).ToNot(HaveOccurred())

			err = fs.MkdirAll("/potato/subdir", 0750)
			Expect(err).ToNot(HaveOccurred())

			fi, err := fs.Stat("/potato")
			Expect(err).ToNot(HaveOccurred())
			Expect(fi.IsDir()).To(BeTrue())
			Expect(fi.Mode()).To(Equal(os.FileMode(0700)))
		})

		It("doesn't allow directories to be made in files", func() {
			err := fs.WriteFileString("/potato", "I am a file")
			Expect(err).ToNot(HaveOccurred())

			err = fs.MkdirAll("/potato/subdir", 0750)
			Expect(err).To(HaveOccurred())

			err = fs.MkdirAll("/potato/subdir/another", 0750)
			Expect(err).To(HaveOccurred())
		})

		It("handles relative paths", func() {
			err := fs.MkdirAll("potato", 0750)
			Expect(err).ToNot(HaveOccurred())

			fi, err := fs.Stat("potato")
			Expect(err).ToNot(HaveOccurred())
			Expect(fi.IsDir()).To(BeTrue())
			Expect(fi.Mode()).To(Equal(os.FileMode(0750)))
		})
	})

	Describe("RemoveAll", func() {
		It("removes the specified file", func() {
			fs.WriteFileString("foobar", "asdfghjk")
			fs.WriteFileString("foobarbaz", "qwertyuio")

			err := fs.RemoveAll("foobar")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobar")).To(BeFalse())
			Expect(fs.FileExists("foobarbaz")).To(BeTrue())

			err = fs.RemoveAll("foobarbaz")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobarbaz")).To(BeFalse())
		})

		It("works with windows drives", func() {
			fs.WriteFileString("D:/env1", "fake-content1")
			Expect(fs.FileExists("D:/env1")).To(BeTrue())

			fs.WriteFileString("C:/env2", "fake-content2")
			Expect(fs.FileExists("C:/env2")).To(BeTrue())
		})

		It("removes the specified dir and the files under it", func() {
			err := fs.MkdirAll("foobarbaz", os.ModePerm)
			Expect(err).ToNot(HaveOccurred())
			err = fs.WriteFileString("foobarbaz/stuff.txt", "asdfghjk")
			Expect(err).ToNot(HaveOccurred())
			err = fs.MkdirAll("foobar", os.ModePerm)
			Expect(err).ToNot(HaveOccurred())
			err = fs.WriteFileString("foobar/stuff.txt", "qwertyuio")
			Expect(err).ToNot(HaveOccurred())

			err = fs.RemoveAll("foobar")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobar")).To(BeFalse())
			Expect(fs.FileExists("foobar/stuff.txt")).To(BeFalse())
			Expect(fs.FileExists("foobarbaz")).To(BeTrue())
			Expect(fs.FileExists("foobarbaz/stuff.txt")).To(BeTrue())

			err = fs.RemoveAll("foobarbaz")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobarbaz")).To(BeFalse())
			Expect(fs.FileExists("foobarbaz/stuff.txt")).To(BeFalse())
		})

		It("removes the specified symlink (but not the file it links to)", func() {
			err := fs.WriteFileString("foobarbaz", "asdfghjk")
			Expect(err).ToNot(HaveOccurred())
			err = fs.Symlink("foobarbaz", "foobar")
			Expect(err).ToNot(HaveOccurred())

			err = fs.RemoveAll("foobarbaz")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobarbaz")).To(BeFalse())
			Expect(fs.FileExists("foobar")).To(BeTrue())

			err = fs.RemoveAll("foobar")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("foobar")).To(BeFalse())
		})

		Context("RemoveAllStub", func() {
			It("calls it and performs its normal behavior as well", func() {
				called := false
				fs.RemoveAllStub = func(path string) error {
					called = true
					return nil
				}
				fs.WriteFileString("foobar", "asdfghjk")

				err := fs.RemoveAll("foobar")
				Expect(err).ToNot(HaveOccurred())

				Expect(fs.FileExists("foobar")).To(BeFalse())
				Expect(called).To(BeTrue())
			})

			It("supports returning an error", func() {
				fs.RemoveAllStub = func(path string) error {
					return errors.New("ERR")
				}
				err := fs.RemoveAll("foobar")
				Expect(err).To(MatchError("ERR"))
			})
		})
	})

	Describe("CopyDir", func() {
		var fixtureFiles = map[string]string{
			"foo.txt":         "asdfghjkl",
			"bar/bar.txt":     "qwertyuio",
			"bar/baz/bar.txt": "zxcvbnm,\nafawg",
		}

		var (
			fixtureDirPath = "fixtures"
		)

		BeforeEach(func() {
			for fixtureFile, contents := range fixtureFiles {
				fs.WriteFileString(filepath.Join(fixtureDirPath, fixtureFile), contents)
			}
		})

		It("recursively copies directory and directory contents", func() {
			srcPath := fixtureDirPath
			tmpDir, err := fs.TempDir("CopyDirTestDir")
			Expect(err).ToNot(HaveOccurred())
			defer fs.RemoveAll(tmpDir)

			destPath := filepath.Join(tmpDir, "dest")

			err = fs.CopyDir(srcPath, destPath)
			Expect(err).ToNot(HaveOccurred())

			for fixtureFile := range fixtureFiles {
				srcContents, err := fs.ReadFile(filepath.Join(srcPath, fixtureFile))
				Expect(err).ToNot(HaveOccurred())

				dstContents, err := fs.ReadFile(filepath.Join(destPath, fixtureFile))
				Expect(err).ToNot(HaveOccurred())

				Expect(srcContents).To(Equal(dstContents), "Copied file does not match source file: '%s", fixtureFile)
			}

			_, err = fs.Stat(destPath)
			Expect(err).ToNot(HaveOccurred())

			err = fs.RemoveAll(tmpDir)
			Expect(err).ToNot(HaveOccurred())
		})

		It("preserves file permissions", func() {
			srcPath, err := fs.TempDir("CopyDirTestSrc")
			Expect(err).ToNot(HaveOccurred())

			readOnly := filepath.Join(srcPath, "readonly.txt")
			err = fs.WriteFileString(readOnly, "readonly")
			Expect(err).ToNot(HaveOccurred())

			err = fs.Chmod(readOnly, 0400)
			Expect(err).ToNot(HaveOccurred())

			dstPath, err := fs.TempDir("CopyDirTestDest")
			Expect(err).ToNot(HaveOccurred())
			defer fs.RemoveAll(dstPath)

			err = fs.CopyDir(srcPath, dstPath)
			Expect(err).ToNot(HaveOccurred())

			fi, err := fs.Stat(filepath.Join(dstPath, "readonly.txt"))
			Expect(err).ToNot(HaveOccurred())

			Expect(fi.Mode()).To(Equal(os.FileMode(0400)))
		})
	})

	Describe("GlobStub", func() {
		It("should allow glob to be replaced with a custom callback", func() {
			fs.GlobStub = func(pattern string) ([]string, error) {
				fs.GlobStub = nil
				return []string{}, errors.New("Oh noes!")
			}
			fs.SetGlob("glob/pattern", []string{"matchingFile1", "matchingFile2"})

			matches, err := fs.Glob("glob/pattern")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Oh noes!"))
			Expect(matches).To(BeEmpty())

			matches, err = fs.Glob("glob/pattern")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(matches)).To(Equal(2))
		})
	})

	Describe("Rename", func() {
		It("renames", func() {
			file, err := fs.OpenFile("foobarbaz", 1, 0600)
			Expect(err).ToNot(HaveOccurred())
			_, err = file.Write([]byte("asdf"))
			Expect(err).ToNot(HaveOccurred())
			err = file.Close()
			Expect(err).ToNot(HaveOccurred())

			err = fs.Chown("foobarbaz", "root:vcap")
			Expect(err).ToNot(HaveOccurred())

			oldStat := fs.GetFileTestStat("foobarbaz")

			err = fs.Rename("foobarbaz", "foobar")
			Expect(err).ToNot(HaveOccurred())

			newStat := fs.GetFileTestStat("foobar")
			Expect(newStat.Content).To(Equal(oldStat.Content))
			Expect(newStat.FileMode).To(Equal(oldStat.FileMode))
			Expect(newStat.FileType).To(Equal(oldStat.FileType))
			Expect(newStat.Username).To(Equal(oldStat.Username))
			Expect(newStat.Groupname).To(Equal(oldStat.Groupname))
			Expect(newStat.Flags).To(Equal(oldStat.Flags))
		})

		It("renames the contents of subdirectories", func() {
			err := fs.MkdirAll("originaldir", 0700)
			Expect(err).ToNot(HaveOccurred())

			err = fs.WriteFileString("originaldir/file.txt", "contents!")
			Expect(err).ToNot(HaveOccurred())

			err = fs.Rename("originaldir", "newdir")
			Expect(err).ToNot(HaveOccurred())

			contents, err := fs.ReadFileString("newdir/file.txt")
			Expect(err).ToNot(HaveOccurred())
			Expect(contents).To(Equal("contents!"))
		})
	})

	Describe("Symlink", func() {
		It("creates", func() {
			err := fs.Symlink("foobarbaz", "foobar")
			Expect(err).ToNot(HaveOccurred())

			stat, err := fs.Lstat("foobar")
			Expect(err).ToNot(HaveOccurred())

			Expect(stat.Mode() & os.ModeSymlink).ToNot(Equal(0))
		})
	})

	Describe("ReadAndFollowLink", func() {
		Context("when the target exists", func() {
			It("returns the target", func() {
				err := fs.WriteFileString("foobarbaz", "asdfghjk")
				Expect(err).ToNot(HaveOccurred())
				err = fs.Symlink("foobarbaz", "foobar")
				Expect(err).ToNot(HaveOccurred())

				targetPath, err := fs.ReadAndFollowLink("foobar")
				Expect(err).ToNot(HaveOccurred())
				Expect(targetPath).To(Equal("foobarbaz"))
			})

			Context("using absolute directory paths", func() {
				It("returns the absolute path of target", func() {
					err := fs.WriteFileString("/tmp/foobarbaz", "asdfghjk")
					Expect(err).ToNot(HaveOccurred())
					err = fs.Symlink("/tmp/foobarbaz", "/tmp/foobar")
					Expect(err).ToNot(HaveOccurred())

					targetPath, err := fs.ReadAndFollowLink("/tmp/foobar")
					Expect(err).ToNot(HaveOccurred())
					absFilepath, err := filepath.Abs("/tmp/foobarbaz")
					Expect(err).ToNot(HaveOccurred())
					Expect(targetPath).To(Equal(absFilepath))
				})
			})
		})

		Context("when the target is located in a parent directory", func() {
			It("returns the target", func() {
				sysRoot := os.Getenv("SYSTEMROOT")
				err := fs.WriteFileString(filepath.Join(sysRoot, "foobarbaz"), "asdfghjk")
				Expect(err).ToNot(HaveOccurred())

				err = fs.Symlink(filepath.Join(sysRoot, "a", "b", "..", "..", "foobarbaz"), "foobar")
				Expect(err).ToNot(HaveOccurred())

				targetPath, err := fs.ReadAndFollowLink("foobar")
				Expect(err).ToNot(HaveOccurred())
				absTarget, err := filepath.Abs(targetPath)
				Expect(err).ToNot(HaveOccurred())
				absExpected, err := filepath.Abs(filepath.Join(sysRoot, "foobarbaz"))
				Expect(err).ToNot(HaveOccurred())

				Expect(absTarget).To(Equal(absExpected))
			})
		})

		Context("when the target file does not exist", func() {
			It("returns an error", func() {
				err := fs.Symlink("non-existant-target", "foobar")
				Expect(err).ToNot(HaveOccurred())

				_, err = fs.ReadAndFollowLink("foobar")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when there are intermediate symlinks", func() {
			It("returns the target", func() {
				err := fs.WriteFileString("foobarbaz", "asdfghjk")
				Expect(err).ToNot(HaveOccurred())

				err = fs.Symlink("foobarbaz", "foobarbazmid")
				Expect(err).ToNot(HaveOccurred())

				err = fs.Symlink("foobarbazmid", "foobar")
				Expect(err).ToNot(HaveOccurred())

				targetPath, err := fs.ReadAndFollowLink("foobar")
				Expect(err).ToNot(HaveOccurred())
				Expect(targetPath).To(Equal("foobarbaz"))
			})
		})
	})

	Describe("Readlink", func() {
		Context("when the given 'link' is a regular file", func() {
			It("returns an error", func() {
				err := fs.WriteFileString("foobar", "notalink")
				Expect(err).ToNot(HaveOccurred())

				_, err = fs.Readlink("foobar")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the link does not exist", func() {
			It("returns an error", func() {
				_, err := fs.Readlink("foobar")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the target path does not exist", func() {
			It("returns the target path without error", func() {
				err := fs.Symlink("foobarTarget", "foobarSymlink")
				Expect(err).ToNot(HaveOccurred())

				targetPath, err := fs.Readlink("foobarSymlink")
				Expect(err).ToNot(HaveOccurred())
				Expect(targetPath).To(Equal("foobarTarget"))
			})
		})

		Context("when the target path exists", func() {
			It("returns the target path without error", func() {
				fs.WriteFileString("foobarTarget", "asdfasdf")
				Expect(fs.FileExists("foobarTarget")).To(Equal(true))

				err := fs.Symlink("foobarTarget", "foobarSymlink")
				Expect(err).ToNot(HaveOccurred())

				targetPath, err := fs.Readlink("foobarSymlink")
				Expect(err).ToNot(HaveOccurred())
				Expect(targetPath).To(Equal("foobarTarget"))
			})

			Context("using absolute directory paths", func() {
				It("returns the absolute path of target", func() {
					err := fs.WriteFileString("/tmp/foobarbaz", "asdfghjk")
					Expect(err).ToNot(HaveOccurred())
					err = fs.Symlink("/tmp/foobarbaz", "/tmp/foobar")
					Expect(err).ToNot(HaveOccurred())

					targetPath, err := fs.Readlink("/tmp/foobar")
					Expect(err).ToNot(HaveOccurred())
					absFilepath, err := filepath.Abs("/tmp/foobarbaz")
					Expect(err).ToNot(HaveOccurred())
					Expect(targetPath).To(Equal(absFilepath))

				})
			})
		})

		Context("when there is an error", func() {
			It("return the error", func() {
				fs.WriteFileString("foobarTarget", "asdfasdf")
				Expect(fs.FileExists("foobarTarget")).To(Equal(true))

				err := fs.Symlink("foobarTarget", "foobarSymlink")
				Expect(err).ToNot(HaveOccurred())

				fs.ReadlinkError = errors.New("can't read link")

				_, err = fs.Readlink("foobarSymlink")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("can't read link"))
			})
		})
	})

	Describe("RegisterReadFileError", func() {
		It("errors when specified path is read", func() {
			fs.WriteFileString("/some/path", "asdfasdf")

			fs.RegisterReadFileError("/some/path", errors.New("read error"))

			_, err := fs.ReadFile("/some/path")
			Expect(err).To(MatchError("read error"))
		})
	})

	Describe("UnregisterReadFileError", func() {
		It("does not throw an error", func() {
			fs.WriteFileString("/some/path", "asdfasdf")

			fs.RegisterReadFileError("/some/path", errors.New("read error"))
			fs.UnregisterReadFileError("/some/path")

			_, err := fs.ReadFile("/some/path")
			Expect(err).ToNot(HaveOccurred())
		})

		Context("When UnregisterReadFileError is called without registering an error", func() {
			It("should not panic or throw an error", func() {
				fs.WriteFileString("/some/path", "asdfasdf")

				fs.UnregisterReadFileError("/some/path")

				_, err := fs.ReadFile("/some/path")
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("ReadFileWithOpts", func() {
		It("reads the file", func() {
			fs.WriteFileQuietly("foo", []byte("hello"))

			writtenContent, err := fs.ReadFileWithOpts("foo", boshsys.ReadOpts{})
			Expect(err).ToNot(HaveOccurred())
			Expect(string(writtenContent)).To(ContainSubstring("hello"))
		})

		It("Records the number of times the method was called", func() {
			fs.WriteFileQuietly("foo", []byte("hello"))
			fs.ReadFileWithOpts("foo", boshsys.ReadOpts{})
			fs.ReadFileWithOpts("foo", boshsys.ReadOpts{Quiet: true})

			Expect(fs.ReadFileWithOptsCallCount).To(Equal(2))
		})
	})

	Describe("WriteFileQuietly", func() {
		It("Writes the file", func() {
			fs.WriteFileQuietly("foo", []byte("hello"))

			writtenContent, err := fs.ReadFileString("foo")
			Expect(err).ToNot(HaveOccurred())
			Expect(writtenContent).To(ContainSubstring("hello"))
		})

		It("Records the number of times the method was called", func() {
			fs.WriteFileQuietly("foo", []byte("hello"))
			fs.WriteFileQuietly("bar", []byte("hello"))

			Expect(fs.WriteFileQuietlyCallCount).To(Equal(2))
		})
	})

	Describe("Ls", func() {
		It("Lists files in a directory", func() {
			err := fs.WriteFile("foo/bar", []byte("hello"))
			Expect(err).NotTo(HaveOccurred())
			err = fs.WriteFile("foo/baz", []byte("hello"))
			Expect(err).NotTo(HaveOccurred())

			results, err := fs.Ls("foo")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(Equal([]string{"foo/bar", "foo/baz"}))
		})
	})

	Describe("Walk", func() {
		It("Recursively traverses a directory", func() {
			err := fs.WriteFile("foo/bar", []byte("hello"))
			Expect(err).NotTo(HaveOccurred())
			err = fs.WriteFile("foobar", []byte("should not be included"))
			Expect(err).NotTo(HaveOccurred())
			err = fs.WriteFile("foo/bam/bang", []byte("hello"))
			Expect(err).NotTo(HaveOccurred())
			err = fs.WriteFile("foo/baz", []byte("hello"))
			Expect(err).NotTo(HaveOccurred())

			expectedFiles := []string{"foo", "foo/bam", "foo/bam/bang", "foo/bar", "foo/baz"}
			actualFiles := []string{}
			// foo/bar/.. makes it necessary to handle path expansion, which should be handled gracefully by the fake filesystem as well.
			err = fs.Walk("foo/bar/..", func(path string, _ os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				actualFiles = append(actualFiles, path)
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(actualFiles).To(Equal(expectedFiles))
		})
	})

	Describe("WriteFile", func() {
		It("Writes the file", func() {
			fs.WriteFile("foo", []byte("hello"))

			writtenContent, err := fs.ReadFileString("foo")
			Expect(err).ToNot(HaveOccurred())
			Expect(writtenContent).To(ContainSubstring("hello"))
		})

		It("Records the number of times the method was called", func() {
			fs.WriteFile("foo", []byte("hello"))
			fs.WriteFile("bar", []byte("hello"))

			Expect(fs.WriteFileCallCount).To(Equal(2))
		})
	})

	Describe("Stat", func() {
		It("errors when symlink targets do not exist", func() {
			err := fs.Symlink("foobarbaz", "foobar")
			Expect(err).ToNot(HaveOccurred())

			_, err = fs.Stat("foobar")
			Expect(err).To(HaveOccurred())
		})

		It("follows symlink target to show its stats", func() {
			err := fs.WriteFileString("foobarbaz", "asdfghjk")
			Expect(err).ToNot(HaveOccurred())

			err = fs.Symlink("foobarbaz", "foobar")
			Expect(err).ToNot(HaveOccurred())

			_, err = fs.Stat("foobar")
			Expect(err).ToNot(HaveOccurred())
		})

		It("allows setting ModTime on a fakefile", func() {
			setModTime := time.Now()

			fakeFile := NewFakeFile("foobar", fs)
			fakeFile.Stats = &FakeFileStats{}
			fakeFile.Stats.ModTime = setModTime

			fs.RegisterOpenFile("foobar", fakeFile)

			fileStat, err := fs.Stat("foobar")
			Expect(err).ToNot(HaveOccurred())
			Expect(fileStat.ModTime()).To(Equal(setModTime))
		})

		It("records the invocation", func() {
			err := fs.WriteFileString("somepath", "some file contents")
			Expect(err).ToNot(HaveOccurred())

			_, err = fs.Stat("somepath")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.StatCallCount).To(Equal(1))
		})
	})

	Describe("StatWithOpts", func() {
		It("records the invocation", func() {
			err := fs.WriteFileString("somepath", "some file contents")
			Expect(err).ToNot(HaveOccurred())
			statOpts := boshsys.StatOpts{Quiet: true}

			_, err = fs.StatWithOpts("somepath", statOpts)
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.StatWithOptsCallCount).To(Equal(1))
		})
	})

	Describe("Lstat", func() {
		It("returns symlink info to a target that does not exist", func() {
			err := fs.Symlink("foobarbaz", "foobar")
			Expect(err).ToNot(HaveOccurred())

			_, err = fs.Lstat("foobar")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns symlink info to a target that exists", func() {
			err := fs.WriteFileString("foobarbaz", "asdfghjk")
			Expect(err).ToNot(HaveOccurred())

			err = fs.Symlink("foobarbaz", "foobar")
			Expect(err).ToNot(HaveOccurred())

			_, err = fs.Lstat("foobar")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("ConvergeFileContents", func() {
		It("converges file contents", func() {
			err := fs.WriteFileString("/file", "content1")
			Expect(err).ToNot(HaveOccurred())

			changed, err := fs.ConvergeFileContents("/file", []byte("content2"))
			Expect(err).ToNot(HaveOccurred())
			Expect(changed).To(BeTrue())

			Expect(fs.ReadFileString("/file")).To(Equal("content2"))

			changed, err = fs.ConvergeFileContents("/file", []byte("content2"))
			Expect(err).ToNot(HaveOccurred())
			Expect(changed).To(BeFalse())

			Expect(fs.ReadFileString("/file")).To(Equal("content2"))
		})

		It("does not converges file contents if it's a dry run", func() {
			err := fs.WriteFileString("/file", "content1")
			Expect(err).ToNot(HaveOccurred())

			changed, err := fs.ConvergeFileContents("/file", []byte("content2"), boshsys.ConvergeFileContentsOpts{DryRun: true})
			Expect(err).ToNot(HaveOccurred())
			Expect(changed).To(BeTrue())

			Expect(fs.ReadFileString("/file")).To(Equal("content1"))

			changed, err = fs.ConvergeFileContents("/file", []byte("content1"), boshsys.ConvergeFileContentsOpts{DryRun: true})
			Expect(err).ToNot(HaveOccurred())
			Expect(changed).To(BeFalse())

			Expect(fs.ReadFileString("/file")).To(Equal("content1"))
		})
	})
})
