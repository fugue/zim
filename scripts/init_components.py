from glob import glob
import os
from string import Template
import sys
from typing import Optional, List


def read_file(fpath) -> List[str]:
    with open(fpath, 'r') as f:
        return f.readlines()


def write_file(fpath, text):
    with open(fpath, 'w') as f:
        f.write(text)


def detect_type(cdir) -> Optional[dict]:
    files = set()
    for fpath in glob(os.path.join(cdir, '*')):
        files.add(os.path.basename(fpath))
    if 'Makefile' not in files:
        return None
    name = None
    stack_name = None
    for line in read_file(os.path.join(cdir, 'Makefile')):
        line = line.strip()
        parts = list(map(str.strip, line.split('=')))
        if len(parts) == 2:
            if parts[0] == 'name':
                name = parts[1]
            if parts[0] == 'stack_name':
                stack_name = parts[1]
    if not name:
        return None
    ctype = None
    if 'requirements.txt' in files:
        ctype = 'python'
    elif 'go.mod' in files:
        ctype = 'go'
    elif 'package.json' in files:
        ctype = 'node'
    elif 'cloudformation.yaml' in files:
        ctype = 'cloudformation'
    else:
        return None
    return dict(type=ctype, name=name, stack_name=stack_name, dir=cdir)


# TODO: sed in here
python_tmpl = Template('''name: $name
kind: python
''')

go_tmpl = Template('''name: $name
kind: go
''')

node_tmpl = Template('''name: $name
kind: node
''')

cfn_tmpl = Template('''name: $name
kind: cloudformation
''')

templates = {
    'go': go_tmpl,
    'python': python_tmpl,
    'cloudformation': cfn_tmpl,
    'node': node_tmpl,
}

skip = {
    'db',
    'domain',
    'envapi',
    'green_zebra',
    'lambda_utils',
    'terraform_schemas',
    'frontend_api',
    'remediation_andon',
    'opa_validations',
}


def write_component_yaml(component: dict):
    text = templates[component['type']].substitute(
        name=component['name'])
    write_file(os.path.join(component['dir'], 'component.yaml'), text)


def main():
    by_type = {}
    skipped = []
    directory = sys.argv[1]
    for cdir in glob(os.path.join(directory, 'src', '*')):
        info = detect_type(cdir)
        if not info:
            skipped.append(cdir)
            continue
        if info['name'] in skip:
            skipped.append(cdir)
            continue
        by_type.setdefault(info['type'], []).append(info)
    for ctype in by_type:
        for item in by_type[ctype]:
            print(item)
            write_component_yaml(item)


if __name__ == '__main__':
    main()
