"""
Shows cache contents
"""
import argparse
import json
import sys
import botocore.session
from botocore.config import Config


def fail_with_error(message):
    print(message, file=sys.stderr)
    sys.exit(1)


def show_status(*args, **kwargs):
    kwargs['file'] = sys.stderr
    print(*args, **kwargs)


def show_json(output):
    print(json.dumps(output, sort_keys=True, indent='  '))


def get_json(s3_client, bucket, key):
    response = s3_client.get_object(Bucket=bucket, Key=key)
    try:
        return json.loads(response['Body'].read().decode())
    finally:
        response['Body'].close()


def longest(rows, column):
    """
    Returns the string length of the longest value in the column.
    Or, if the column name is longer than any value, return its length instead.
    """
    col_max = max([len(str(row[column])) for row in rows])
    return max(len(column), col_max)


def show_table(rows, columns, separator=' | ', header=True):
    """
    Prints a table with the given rows and column names to stdout
    """
    # Calculate column widths needed for data and header strings
    column_widths = [longest(rows, column) for column in columns]
    column_formats = ['%%-%ds' % width for width in column_widths]
    horiz_line = '=' * (sum(column_widths) + len(separator) * (len(columns)-1))
    # Print table header
    if header:
        print(horiz_line)
        column_headers = [column_formats[i] % columns[i].upper()
                          for i, _ in enumerate(columns)]
        print(separator.join(column_headers))
        print(horiz_line)
    # Print table rows
    for row in rows:
        column_values = [column_formats[i] % row[col]
                         for i, col in enumerate(columns)]
        print(separator.join(column_values))


def to_json(obj):
    return obj


def list_objects(client, bucket):
    objects = []
    token = None
    while True:
        kwargs = {
            'Bucket': bucket,
            'MaxKeys': 1000,
            'Prefix': 'cache',
        }
        if token:
            kwargs['ContinuationToken'] = token
        resp = client.list_objects_v2(**kwargs)
        objects.extend(resp['Contents'])
        if resp['IsTruncated']:
            token = resp['NextContinuationToken']
        else:
            break
    return objects


def get_cache_keys(objects):
    cache_keys = []
    for obj in objects:
        s3_key = obj['Key']
        parts = s3_key.split('/')
        if len(parts) != 2:
            raise ValueError(f'Unexpected S3 key: {s3_key}')
        if parts[1].endswith('.json'):
            cache_key = parts[1][:-5]
            cache_keys.append((s3_key, cache_key))
    return cache_keys


def main():
    parser = argparse.ArgumentParser(description='Show Zim Cache')
    parser.add_argument('--bucket', help='S3 Bucket Name')
    args = parser.parse_args()

    config = Config(retries=dict(max_attempts=10))
    session = botocore.session.get_session()
    client = session.create_client('s3', config=config)

    keys = get_cache_keys(list_objects(client, args.bucket))
    show_status(f'Found {len(keys)} cache keys')

    keys_json = {}
    for s3_key, cache_key in keys:
        js = get_json(client, args.bucket, s3_key)
        keys_json[cache_key] = dict(
            key=cache_key,
            project=js['project'],
            component=js['component'],
            rule=js['rule'],
            image=js['image'],
        )
        if len(keys_json) % 100 == 0:
            show_status(f'Retrieved {len(keys_json)} keys')

    rows = [to_json(key) for key in keys_json.values()]
    if not rows:
        print('No rows')
        return
    print(rows[0])

    show_table(rows, columns=['key', 'project', 'component', 'rule', 'image'])


if __name__ == '__main__':
    main()
