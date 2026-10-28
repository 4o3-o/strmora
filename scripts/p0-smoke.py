#!/usr/bin/env python3
"""P0 HTTP simulation; private configuration and reports stay under ignored data/."""
import argparse
import datetime as dt
import json
from pathlib import Path
import time
import urllib.error
import urllib.parse
import urllib.request
from zoneinfo import ZoneInfo

ROOT = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser()
parser.add_argument('--base-url', default='http://127.0.0.1:20000')
parser.add_argument('--config', type=Path, default=ROOT/'data/p0-local/config.json')
parser.add_argument('--api-user-agent', default='curl/8.5.0', help='explicit API probe UA; real clients tested separately')
parser.add_argument('--probe-media', action='store_true', help='HEAD + two 1KiB range probes, spaced 3s apart')
parser.add_argument('--media-user-agent', default='curl/8.5.0', help='explicit upstream probe UA; does not change player traffic')
args = parser.parse_args()
base = args.base_url.rstrip('/')
parsed = urllib.parse.urlsplit(base)
if parsed.scheme not in ('http', 'https') or not parsed.netloc or parsed.username or parsed.query or parsed.fragment:
    raise SystemExit('Use an HTTP(S) base URL without embedded credentials, query, or fragment')
config = json.loads(args.config.read_text())
rows = []

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *a, **kw):
        return None

opener = urllib.request.build_opener(NoRedirect)

def request(method, path, body=None, token=None, headers=None):
    h = {'Content-Type': 'application/json', 'User-Agent': args.api_user_agent}
    if token:
        h['X-Emby-Token'] = token
    h.update(headers or {})
    r = urllib.request.Request(base+path, method=method, headers=h,
                               data=json.dumps(body).encode() if body is not None else None)
    try:
        response = opener.open(r, timeout=15)
    except urllib.error.HTTPError as e:
        response = e
    with response:
        return response.status, response.headers, response.read(2*1024*1024)

def check(name, ok, **details):
    rows.append({'check': name, 'passed': bool(ok), **details})
    print(('PASS ' if ok else 'FAIL ')+name)

def obj(b):
    return json.loads(b)

try:
    status, _, _ = request('GET', '/healthz')
    check('health', status == 200)
    status, _, _ = request('GET', '/emby/System/Info/Public')
    check('public_info_with_emby_prefix', status == 200)
    status, _, _ = request('GET', '/Users/Me')
    check('anonymous_rejected', status == 401)
    status, _, b = request('POST', '/Users/AuthenticateByName', {
        'Username': config['STRMORA_P0_USER'], 'Pw': config['STRMORA_P0_PASSWORD']})
    check('password_login', status == 200)
    token = obj(b)['AccessToken']
    status, _, b = request('GET', '/Users/user-p0/Views', token=token)
    check('two_libraries', status == 200 and obj(b)['TotalRecordCount'] == 2)
    status, _, b = request('GET', '/Users/user-p0/Items?IncludeItemTypes=Movie', token=token)
    check('two_movies', status == 200 and obj(b)['TotalRecordCount'] == 2)
    status, _, b = request('GET', '/Shows/series-p0/Seasons', token=token)
    seasons = obj(b)['Items']
    check('two_seasons_and_alias', status == 200 and len(seasons) == 2 and seasons[1]['Name'] == '1997' and seasons[1]['IndexNumber'] == 2)
    status, _, b = request('POST', '/Items/movie-versions/PlaybackInfo', {}, token)
    check('two_playback_sources', status == 200 and len(obj(b)['MediaSources']) == 2)
    target = None
    for item in ['movie-p0', 'movie-versions', 'episode-1', 'episode-2']:
        status, h, _ = request('GET', '/Videos/'+item+'/stream', token=token)
        check('strm_302_'+item, status == 302 and urllib.parse.urlsplit(h.get('Location', '')).scheme in ('http', 'https'))
        if item == 'movie-p0':
            target = h.get('Location')
    status, _, _ = request('HEAD', '/Videos/movie-p0/stream', token=token)
    check('head_302', status == 302)
    status, h, b = request('GET', '/Videos/movie-p0/movie-p0-source/Subtitles/0/Stream.srt', token=token)
    check('subtitle_download', status == 200 and b'Strmora P0' in b)
    status, h, b = request('GET', '/Videos/movie-p0/movie-p0-source/Subtitles/0/Stream.srt', token=token, headers={'Range': 'bytes=0-15'})
    check('subtitle_range', status == 206 and len(b) == 16)
    status, h, b = request('GET', '/Items/movie-p0/Images/Primary', token=token)
    check('placeholder_image', status == 200 and b.startswith(b'\x89PNG'))
    for event in ['Playing', 'Playing/Progress', 'Playing/Stopped']:
        status, _, _ = request('POST', '/Sessions/'+event, {'ItemId': 'movie-p0', 'PositionTicks': 120000000, 'PlaySessionId': 'p0-local-simulation'}, token)
        check('event_'+event, status == 204)
    status, _, b = request('GET', '/Users/user-p0/Items/movie-p0', token=token)
    check('saved_12s_progress', status == 200 and obj(b)['UserData']['PlaybackPositionTicks'] == 120000000)
    status, _, _ = request('POST', '/Sessions/Playing/Progress', {'ItemId': 'movie-p0'}, token)
    check('missing_position_rejected', status == 400)
    status, _, _ = request('GET', '/Users/another-user/Items', token=token)
    check('other_user_rejected', status == 403)
    if args.probe_media and target:
        # Bounded reads: do not download the movie even if upstream ignores Range.
        for method, rang in [('HEAD', None), ('GET', 'bytes=0-1023'), ('GET', 'bytes=1048576-1049599')]:
            time.sleep(3)
            headers = {'Accept-Encoding': 'identity', 'User-Agent': args.media_user_agent}
            if rang:
                headers['Range'] = rang
            r = urllib.request.Request(target, method=method, headers=headers)
            try:
                with urllib.request.urlopen(r, timeout=30) as response:
                    payload = response.read(1024) if method == 'GET' else b''
                    ok = response.status == (206 if rang else 200)
                    if rang:
                        ok = ok and response.headers.get('Content-Range', '').startswith('bytes '+rang.split('=')[1]+'/') and len(payload) == 1024
                    check('upstream_'+(rang or method), ok, status=response.status,
                          content_type=response.headers.get('Content-Type'),
                          content_range=response.headers.get('Content-Range'), bytes_read=len(payload))
            except Exception as e:
                check('upstream_'+(rang or method), False, error_type=type(e).__name__, status=getattr(e, 'code', None))
except Exception as e:
    check('simulation_exception', False, error_type=type(e).__name__)
finally:
    timestamp = dt.datetime.now(ZoneInfo('Asia/Shanghai'))
    out = ROOT/'data/p0-local'/('smoke-'+timestamp.strftime('%Y%m%d-%H%M%S')+'.json')
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps({'time': timestamp.isoformat(), 'base_url': base, 'checks': rows,
                               'real_clients': 'not tested', 'api_user_agent': args.api_user_agent, 'media_user_agent': args.media_user_agent}, ensure_ascii=False, indent=2)+'\n')
    out.chmod(0o600)
    print('Report: '+str(out))
raise SystemExit(0 if rows and all(x['passed'] for x in rows) else 1)
