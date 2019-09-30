# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# isulad-tools is licensed under the Mulan PSL v1.
# You can use this software according to the terms and conditions of the Mulan PSL v1.
# You may obtain a copy of Mulan PSL v1 at:
#    http://license.coscl.org.cn/MulanPSL
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v1 for more details.
# Description: make install
# Author: zhangwei
# Create: 2018-01-18

#!/bin/bash

HOOK_DIR=/var/lib/lcrd/hooks
ISULAD_TOOLS_DIR=/usr/local/bin
ISULAD_TOOLS_WRAPPER="/lib/udev"
HOOK_SPEC="/etc/isulad-tools"

echo "Hooks will be installed to $HOOK_DIR"
echo "isulad_tools will be installed to $ISULAD_TOOLS_DIR"

mkdir -p -m 0700 ${HOOK_DIR}
mkdir -p -m 0750 ${ISULAD_TOOLS_DIR}
mkdir -p -m 0750 ${ISULAD_TOOLS_WRAPPER}
mkdir -p -m 0750 ${HOOK_SPEC}

install -m 0755 -p ../build/*-hooks ${HOOK_DIR}
install -m 0755 -p ../build/isulad-tools ${ISULAD_TOOLS_DIR}
install -m 0750 isulad-tools_wrapper  ${ISULAD_TOOLS_WRAPPER}/isulad-tools_wrapper

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
