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

  const latest = new URL('/argoproj-labs/argocd-image-updater/releases/latest', base);

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
    throw new Error('Could not determine latest ArgoCD Image Updater version');
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
      return 'amd64';
    default:
      throw new Error(`unsupported architecture: ${arch}`);
  }
}

function realName(name, platform, arch) {
  switch (platform) {
    case 'win32':
      return `${name}-${platform}.exe`;
    default:
      return `${name}-${platform}_${arch}`;
  }
}

async function setup() {
  try {
    const server = core.getInput('argocd-server');
    const token = core.getInput('argocd-token');

    const version = await realVersion(core.getInput('argocd-image-updater-version'));
    const platform = realPlatform(process.platform);
    const arch = realArch(process.arch);
    const name = realName('argocd-image-updater', platform, arch);

    const execOptions = { silent: true };

    core.info(`Setup ArgoCD Image Updater ${version}: ${platform} ${arch}`);
    let cachedPath = cache.find(name, 'argocd-image-updater', version);
    if (!cachedPath) {
      const url = new URL(`argoproj-labs/argocd-image-updater/releases/download/v${version}/${name}`, base);

      core.info(`Downloading ArgoCD Image Updater ${version} from ${url}`);
      const toolPath = await cache.downloadTool(url);

      // set the execute bit on the downloaded binary
      await exec.exec('chmod', ['+x', toolPath], execOptions);
      cachedPath = await cache.cacheFile(toolPath, name, 'argocd-image-updater', version);
    }

    // symlink binary name to `argocd-image-updater` for convenience
    await exec.exec('ln', ['-s', `${cachedPath}/${name}`, `${cachedPath}/argocd-image-updater`], execOptions);

    core.info(`ArgoCD Image Updater path is ${cachedPath}`);
    core.addPath(cachedPath);

    if (server) {
      core.exportVariable('ARGOCD_SERVER', server);
    }

    if (token) {
      core.exportVariable('ARGOCD_TOKEN', token);
    }

    core.setOutput('argocd-image-updater-version', version);
  } catch (e) {
    core.setFailed(e.message);
  }
}

setup();
