#!/usr/bin/python
# -*- coding: utf-8 -*-

import re
import os

def version_symbol(ver):
    os.system("cd ./curl && git checkout {}".format(ver))
    opts = []
    codes = []
    infos = []
    pattern = re.compile(r'CINIT\((.*?), (LONG|OBJECTPOINT|FUNCTIONPOINT|OFF_T), (\d+)\)')
    pattern2 = re.compile('^\s+(CURLE_[A-Z_0-9]+),')
    pattern3 = re.compile('^\s+(CURLINFO_[A-Z_0-9]+)\s+=')
    for line in open("./curl/include/curl/curl.h"):
        match = pattern.findall(line)
        if match:
            opts.append("CURLOPT_" + match[0][0])
        if line.startswith('#define CURLOPT_'):
            o = line.split()
            opts.append(o[1])

        match = pattern2.findall(line)
        if match:
            codes.append(match[0])

        if line.startswith('#define CURLE_'):
            c = line.split()
            codes.append(c[1])

        match = pattern3.findall(line)
        if match:
            infos.append(match[0])

        if line.startswith('#define CURLINFO_'):
            i = line.split()
            if '0x' not in i[2]:    # :(
                infos.append(i[1])

    return opts, codes, infos


versions = """
curl-7_10_1
curl-7_10_2
curl-7_10_3
curl-7_10_4
curl-7_10_5
curl-7_10_6
curl-7_10_7
curl-7_10_8
curl-7_11_0
curl-7_11_1
curl-7_11_2
curl-7_12_0
curl-7_12_1
curl-7_12_2
curl-7_12_3
curl-7_13_0
curl-7_13_1
curl-7_13_2
curl-7_14_0
curl-7_14_1
curl-7_15_0
curl-7_15_1
curl-7_15_2
curl-7_15_3
curl-7_15_4
curl-7_15_5
curl-7_16_0
curl-7_16_1
curl-7_16_2
curl-7_16_3
curl-7_16_4
curl-7_17_0
curl-7_17_1
curl-7_18_0
curl-7_18_1
curl-7_18_2
curl-7_19_0
curl-7_19_1
curl-7_19_2
curl-7_19_3
curl-7_19_4
curl-7_19_5
curl-7_19_6
curl-7_19_7
curl-7_20_0
curl-7_20_1
curl-7_21_0
curl-7_21_1
curl-7_21_2
curl-7_21_3
curl-7_21_4
curl-7_21_5
curl-7_21_6
curl-7_21_7
curl-7_22_0
curl-7_23_0
curl-7_23_1
curl-7_24_0
curl-7_25_0
curl-7_26_0
curl-7_27_0
curl-7_28_0
curl-7_28_1
curl-7_29_0
curl-7_30_0
curl-7_31_0
curl-7_32_0
curl-7_33_0
curl-7_34_0
curl-7_35_0
curl-7_36_0""".split()[::-1]

last = version_symbol("master")

template = """
/* generated by compatgen.py */
#include<curl/curl.h>


"""

result = [template]
result_tail = ["/* generated ends */\n"]
if __name__ == '__main__':
    for ver in versions:
        minor, patch = map(int, ver.split("_")[-2:])

        opts, codes, infos = curr = version_symbol(ver)

        for o in last[0]:
            if o not in opts:
                result.append("#define {} 0".format(o)) # 0 for nil option
        for c in last[1]:
            if c not in codes:
                result.append("#define {} -1".format(c)) # -1 for error
        for i in last[2]:
            if i not in infos:
                result.append("#define {} 0".format(i)) # 0 for nil

        result.append("#if (LIBCURL_VERSION_MINOR == {} && LIBCURL_VERSION_PATCH < {}) || LIBCURL_VERSION_MINOR < {} ".format(minor, patch, minor))

        result_tail.insert(0, "#endif /* 7.{}.{} */".format(minor, patch))

        last = curr

result.append("#error your version is TOOOOOOOO low")

result.extend(result_tail)

with open("./compat.h", 'w') as fp:
    fp.write('\n'.join(result))
