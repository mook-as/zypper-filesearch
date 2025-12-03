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

# Please submit bugfixes or comments via
# https://github.com/mook-as/zypper-filesearch/issues


Name:           zypper-filesearch
Version:        0.0.1
Release:        0
Summary:        Zypper plugin to search by file path
License:        GPL-2.0-or-later
Group:          System/Packages
URL:            https://github.com/mook-as/zypper-filesearch
Source:         https://github.com/mook-as/zypper-filesearch/archive/refs/tags/%{version}.tar.gz#/%{name}-%{version}.tar.gz
Source1:        vendor.tar.zst
BuildRequires:  fdupes
BuildRequires:  golang-packaging
BuildRequires:  sqlite3-devel
BuildRequires:  zstd

%description
Zypper plugin that searches for packages that contain a given file pattern.

%prep
%autosetup -p1 -a1

%build
go build -mod=vendor -buildmode=pie
go tool go-md2man -in=zypper-file-search.1.md -out=zypper-file-search.1
go tool go-md2man -in=zypper-file-list.1.md -out=zypper-file-list.1

%install
install -D --mode=0755 --strip %{name} %{buildroot}%{_bindir}/zypper-file-search
install -D --mode=0755 --strip %{name} %{buildroot}%{_bindir}/zypper-file-list
install -D --mode=0644 zypper-file-search.1 %{buildroot}%{_mandir}/man1/zypper-file-search.1
install -D --mode=0644 zypper-file-list.1 %{buildroot}%{_mandir}/man1/zypper-file-list.1
%fdupes %{buildroot}%{_bindir}

%files
%license LICENSE.txt GPL-2.0.txt
%doc README.md
%{_bindir}/zypper-file-search
%{_bindir}/zypper-file-list
%doc %{_mandir}/man1/zypper-file-search.1%{?ext_man}
%doc %{_mandir}/man1/zypper-file-list.1%{?ext_man}

%changelog
