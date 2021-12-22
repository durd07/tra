from flask import Flask, json, request

skey_result = [{"skey": "12345", "nodes": [{"nodeid": 1, "ip": "169.254.0.87"}]}]

api = Flask(__name__)


@api.route('/SIP/INT/nodes', methods=['GET'])
def skey():
    skey = request.args.get('skey')
    if skey == '12345':
        return json.dumps(skey_result)
    else:
        return json.dumps([{"skey": "12345", "nodes": []}])

if __name__ == '__main__':
    api.run(port=15810)
