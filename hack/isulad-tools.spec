#Global macro or variabl
%define  debug_package %{nil}

#Basic Information
Name:           isulad-tools
Version:        v0.9
Release:        34
Summary:        isulad tools for IT, work with iSulad
License:        Mulan PSL v1
URL:            https://gitee.com/src-openeuler/iSulad-tools
Source0:        %{name}-src.tar.gz
BuildRoot:      %{_tmppath}/%{name}-root

#Dependency
BuildRequires: glibc-static 
BuildRequires: golang > 1.6
Requires: iSulad
Requires: util-linux

%description
This is isulad tools, to make it work, you need a isulad and util-linux

#Build sections
%prep
%setup -q -c -n src/isula.org/isulad-tools

%build
make init && make

%install
HOOK_DIR=$RPM_BUILD_ROOT/var/lib/lcrd/hooks
ISULAD_TOOLS_DIR=$RPM_BUILD_ROOT/usr/local/bin
ISULAD_TOOLS_WRAPPER=$RPM_BUILD_ROOT/lib/udev

mkdir -p -m 0700 ${HOOK_DIR}
mkdir -p -m 0750 ${ISULAD_TOOLS_DIR}
mkdir -p -m 0750 ${ISULAD_TOOLS_WRAPPER}

install -m 0750 build/isulad-hooks ${HOOK_DIR}
install -m 0750 build/isulad-tools ${ISULAD_TOOLS_DIR}
install -m 0750 hack/isulad-tools_wrapper  ${ISULAD_TOOLS_WRAPPER}/isulad-tools_wrapper

#Install and uninstall scripts
%pre

%preun

%post
GRAPH=`lcrc info | grep -Eo "iSulad Root Dir:.+" | grep -Eo "\/.*"` 
if [ "$GRAPH" == "" ]; then
    GRAPH="/var/lib/lcrd"
fi

if [[ "$GRAPH" != "/var/lib/lcrd" ]]; then
    mkdir -p -m 0550 $GRAPH/hooks
    install -m 0550 -p /var/lib/lcrd/hooks/isulad-hooks $GRAPH/hooks

    echo
    echo "=================== WARNING! ================================================"
    echo " 'iSulad Root Dir' is $GRAPH, move /var/lib/lcrd/hooks/isulad-hooks to  $GRAPH/hooks"
    echo "============================================================================="
    echo
fi
HOOK_SPEC=/etc/isulad-tools
HOOK_DIR=${GRAPH}/hooks
mkdir -p -m 0750 ${HOOK_SPEC}
mkdir -p -m 0550 ${HOOK_DIR}
cat << EOF > ${HOOK_SPEC}/hookspec.json
{
        "prestart": [
        {
                "path": "${HOOK_DIR}/isulad-hooks",
                "args": ["isulad-hooks", "--state", "prestart"],
                "env": []
        }
        ],
        "poststart":[
        {
                "path": "${HOOK_DIR}/isulad-hooks",
                "args": ["isulad-hooks", "--state", "poststart"],
                "env": []
        }
	],
        "poststop":[
        {
                "path": "${HOOK_DIR}/isulad-hooks",
                "args": ["isulad-hooks", "--state", "poststop"],
                "env": []
        }
	]
}
EOF
chmod 0640 ${HOOK_SPEC}/hookspec.json

%postun

#Files list
%files
%defattr(0550,root,root,0550)
/usr/local/bin/isulad-tools
%attr(0550,root,root) /var/lib/lcrd/hooks
%attr(0550,root,root) /var/lib/lcrd/hooks/isulad-hooks
%attr(0550,root,root) /lib/udev/isulad-tools_wrapper


#Clean section
%clean 
rm -rfv %{buildroot}

%changelog
* Tue Dec 26 2019 Zhangsong<zhangsong34@huawei.com> - 0.9.34
- Type:enhancement
- ID:NA
- SUG:restart
- DESC:remove securec functions

* Tue Dec 11 2018 Zhangsong<zhangsong34@huawei.com> - 0.9.9-1
- Type:enhancement
- ID:NA
- SUG:restart
- DESC:fix multiple add-device process to one container

* Tue Dec 04 2018 Zhangsong<zhangsong34@huawei.com> - 0.9.8-1
- Type:enhancement
- ID:NA
- SUG:restart
- DESC:rebuild patches for isulad-tools

* Fri Nov 02 2018 Zhangsong<zhangsong34@huawei.com> - 0.9.7-1
- Type:enhancement
- ID:NA
- SUG:restart
- DESC:add compat hook state struct for old version

* Mon Sep 17 2018 jingrui<jingrui@huawei.com> - 0.9.4-2
- Type:enhancement
- ID:NA
- SUG:restart
- DESC:remove ns-change commands

* Mon Sep 10 2018 wangfengtu<wangfengtu@huawei.com> - 0.9.4-1
- Type:enhancement
- ID:NA
- SUG:restart
- DESC:fix remove mountpoint while have not umount mountpoint

* Thu Jun 7 2018 Liruilin<liruilin4@huawei.com> - 0.9.2-1
- Type:enhancement
- ID:NA
- SUG:restart
- DESC:add poststart for enable-ns-change

* Thu Mar 15 2018 Caoruidong<caoruidong@huawei.com> - 0.9.1-2
- Type:enhancement
- ID:NA
- SUG:restart
- DESC:add require util-linux version

* Sat Feb 3 2018 Tanzhe<tanzhe@huawei.com> - 0.9.1-1
- Type:enhancement
- ID:NA
- SUG:restart
- DESC:add require docker-engine version

* Fri Jan 6 2017 ShenPeng <shenpeng11@huawei.com>
- enable compiling on OBS
