// Copyright 2021 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rust

import (
	"android/soong/android"
	"android/soong/cc"

	"github.com/google/blueprint/proptools"
)

const (
	snapshotRlibSuffix = "_rlib."
)

type snapshotLibraryDecorator struct {
	cc.BaseSnapshotDecorator
	*libraryDecorator
	properties          cc.SnapshotLibraryProperties
	sanitizerProperties struct {
		CfiEnabled bool `blueprint:"mutated"`

		// Library flags for cfi variant.
		Cfi cc.SnapshotLibraryProperties `android:"arch_variant"`
	}
}

func init() {
	registerRustSnapshotModules(android.InitRegistrationContext)
}

func registerRustSnapshotModules(ctx android.RegistrationContext) {
	cc.VendorSnapshotImageSingleton.RegisterAdditionalModule(ctx,
		"vendor_snapshot_rlib", VendorSnapshotRlibFactory)
	cc.RecoverySnapshotImageSingleton.RegisterAdditionalModule(ctx,
		"recovery_snapshot_rlib", RecoverySnapshotRlibFactory)
	cc.RamdiskSnapshotImageSingleton.RegisterAdditionalModule(ctx,
		"ramdisk_snapshot_rlib", RamdiskSnapshotRlibFactory)
}

func snapshotLibraryFactory(image cc.SnapshotImage, moduleSuffix string) (*Module, *snapshotLibraryDecorator) {
	module, library := NewRustLibrary(android.DeviceSupported)

	module.sanitize = nil
	library.stripper.StripProperties.Strip.None = proptools.BoolPtr(true)

	prebuilt := &snapshotLibraryDecorator{
		libraryDecorator: library,
	}

	module.compiler = prebuilt

	prebuilt.Init(module, image, moduleSuffix)
	module.AddProperties(
		&prebuilt.properties,
		&prebuilt.sanitizerProperties,
	)

	return module, prebuilt
}

func (library *snapshotLibraryDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) android.Path {
	var variant string
	if library.static() {
		variant = cc.SnapshotStaticSuffix
	} else if library.shared() {
		variant = cc.SnapshotSharedSuffix
	} else if library.rlib() {
		variant = cc.SnapshotRlibSuffix
	}

	if !library.dylib() {
		// TODO(184042776): Remove this check when dylibs are supported in snapshots.
		library.SetSnapshotAndroidMkSuffix(ctx, variant)
	}

	if !library.MatchesWithDevice(ctx.DeviceConfig()) {
		return nil
	}
	outputFile := android.PathForModuleSrc(ctx, *library.properties.Src)
	library.unstrippedOutputFile = outputFile
	return outputFile
}

func (library *snapshotLibraryDecorator) rustdoc(ctx ModuleContext, flags Flags, deps PathDeps) android.OptionalPath {
	return android.OptionalPath{}
}

// vendor_snapshot_rlib is a special prebuilt rlib library which is auto-generated by
// development/vendor_snapshot/update.py. As a part of vendor snapshot, vendor_snapshot_rlib
// overrides the vendor variant of the rust rlib library with the same name, if BOARD_VNDK_VERSION
// is set.
func VendorSnapshotRlibFactory() android.Module {
	module, prebuilt := snapshotLibraryFactory(cc.VendorSnapshotImageSingleton, cc.SnapshotRlibSuffix)
	prebuilt.libraryDecorator.BuildOnlyRlib()
	prebuilt.libraryDecorator.setNoStdlibs()
	return module.Init()
}

func RecoverySnapshotRlibFactory() android.Module {
	module, prebuilt := snapshotLibraryFactory(cc.RecoverySnapshotImageSingleton, cc.SnapshotRlibSuffix)
	prebuilt.libraryDecorator.BuildOnlyRlib()
	prebuilt.libraryDecorator.setNoStdlibs()
	return module.Init()
}

func RamdiskSnapshotRlibFactory() android.Module {
	module, prebuilt := snapshotLibraryFactory(cc.RamdiskSnapshotImageSingleton, cc.SnapshotRlibSuffix)
	prebuilt.libraryDecorator.BuildOnlyRlib()
	prebuilt.libraryDecorator.setNoStdlibs()
	return module.Init()
}

func (library *snapshotLibraryDecorator) MatchesWithDevice(config android.DeviceConfig) bool {
	arches := config.Arches()
	if len(arches) == 0 || arches[0].ArchType.String() != library.Arch() {
		return false
	}
	if library.properties.Src == nil {
		return false
	}
	return true
}

func (library *snapshotLibraryDecorator) IsSnapshotPrebuilt() bool {
	return true
}

var _ cc.SnapshotInterface = (*snapshotLibraryDecorator)(nil)
