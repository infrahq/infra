const path = require('path');

const core = require('@actions/core');
const exec = require('@actions/exec');
const http = require('@actions/http-client');
const cache = require('@actions/tool-cache');

const base = new URL('https://github.com/');

const HttpRedirectCodes = [
  http.HttpCodes.MovedPermanently,
  http.HttpCodes.ResourceMoved,
  http.HttpCodes.SeeOther,
  http.HttpCodes.TemporaryRedirect,
  http.HttpCodes.PermanentRedirect,
];

async function realVersion(version) {
  if (version !== 'latest') {
    return version;
  }

  const latest = new URL('/infrahq/infra/releases/latest', base);

  const client = new http.HttpClient('', '', { allowRedirects: false });
  const res = await client.head(latest);

  if (!HttpRedirectCodes.includes(res.message.statusCode)) {
    throw new Error('Did not get expected redirect');
  }

  const location = new URL(res.message.headers.location);
  let newVersion = path.basename(location.pathname);

  if (newVersion[0] === 'v') {
    newVersion = newVersion.substring(1);
  }

  if (newVersion === 'latest') {
    throw new Error('Could not determine latest Infra version');
  }

  return newVersion;
}

function realPlatform(platform) {
  switch (platform) {
    case 'darwin':
    case 'linux':
      return platform;
    case 'win32':
      return 'windows';
    default:
      throw new Error(`unsupported platform: ${platform}`);
  }
}

function realArch(arch) {
  switch (arch) {
    case 'arm64':
      return arch;
    case 'x64':
      return 'x86_64';
    default:
      throw new Error(`unsupported architecture: ${arch}`);
  }
}

async function setup() {
  try {
    const server = core.getInput('infra-server');
    const accessKey = core.getInput('infra-access-key');
    const destination = core.getInput('infra-destination');

    if (destination && !accessKey) {
      throw new Error('Cannot set up destination without access key.');
    }

    const version = await realVersion(core.getInput('infra-version'));
    const platform = realPlatform(process.platform);
    const arch = realArch(process.arch);
    const name = ['infra', version, platform, arch].join('_');

    const execOptions = { silent: true };

    core.info(`Setup Infra ${version}: ${platform} ${arch}`);
    let cachedPath = cache.find(name, 'infra', version);
    if (!cachedPath) {
      const url = new URL(`infrahq/infra/releases/download/v${version}/${name}.zip`, base);

      core.info(`Downloading Infra ${version} from ${url}`);
      const zipPath = await cache.downloadTool(url);
      const toolPath = await cache.extractZip(zipPath);

      cachedPath = await cache.cacheDir(toolPath, 'infra', version);
    }

    core.info(`Infra path is ${cachedPath}`);
    core.addPath(cachedPath);

    if (server) {
      core.exportVariable('INFRA_SERVER', server);
    }

    if (accessKey) {
      core.exportVariable('INFRA_ACCESS_KEY', accessKey);
    }

    if (destination) {
      const skipTLSVerify = core.getInput('skip-tls-verify');
      core.info(`Setting kubectl context to ${destination}`);
      await exec.exec('infra', ['login', skipTLSVerify ? '--skip-tls-verify' : ''], execOptions);
      await exec.exec('infra', ['use', destination], execOptions);
    }

    core.setOutput('infra-version', version);
  } catch (e) {
    core.setFailed(e.message);
  }
}

setup();
