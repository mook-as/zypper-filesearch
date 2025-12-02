#
# spec file for zypper-filesearch
#
# Copyright (c) 2025 SUSE LLC
#
# All modifications and additions to the file contributed by third parties
# remain the property of their copyright owners, unless otherwise agreed
# upon. The license for this file, and modifications and additions to the
# file, is the same license as for the pristine package itself (unless the
# license for the pristine package is not an Open Source License, in which
# case the license is the MIT License). An "Open Source License" is a
# license that conforms to the Open Source Definition (Version 1.9)
# published by the Open Source Initiative.

# Please submit bugfixes or comments via https://bugs.opensuse.org/
#


Name:           zypper-filesearch
Version:        0.0.1
Release:        0
Summary:        Zypper plugin to search by file path
License:        GPL-2.0-or-later
Group:          System/Packages
URL:            https://github.com/mook-as/zypper-filesearch
Source:         https://github.com/mook-as/zypper-filesearch/archive/refs/tags/%{version}.tar.gz#/%{name}-%{version}.tar.gz
Source1:        vendor.tar.zst
BuildRequires:  golang-packaging
BuildRequires:  sqlite3-devel
BuildRequires:  zstd

%description
Zypper plugin that searches for packages that contain a given file pattern.

%prep
%autosetup -p1 -a1

%build
go build \
   -mod=vendor \
   -buildmode=pie

%install
install -D -m0755 %{name} %{buildroot}%{_bindir}/%{name}
#install -D -m0644 man/%{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1

%files
%license LICENSE.txt GPL-2.0.txt
%doc README.md
%{_bindir}/%{name}
#%{_mandir}/man1/%{name}.1%{?ext_man}

%changelog
