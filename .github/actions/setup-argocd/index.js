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

async function realVersion(project, version) {
  if (version !== 'latest') {
    return version;
  }

  const latest = new URL(`/${project}/releases/latest`, base);

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
  switch (name) {
    case 'argocd':
      return [name, platform, arch].join('-');
    case 'argocd-image-updater':
      return platform === 'windows' ? `${name}-${platform}_${arch}.exe` : `${name}-${platform}_${arch}`;
    default:
      throw new Error(`unsupported tool: ${name}`);
  }
}

function realProject(name) {
  switch (name) {
    case 'argocd':
      return 'argoproj/argo-cd';
    case 'argocd-image-updater':
      return 'argoproj-labs/argocd-image-updater';
    default:
      throw new Error(`unsupported tool: ${name}`);
  }
}

async function setup() {
  try {
    const tools = core.getMultilineInput('argocd-tools');
    const platform = realPlatform(process.platform);
    const arch = realArch(process.arch);

    const execOptions = { silent: true };

    const installed = [];

    tools.forEach(async (item) => {
      const parts = item.split('=');
      const name = realName(parts[0], platform, arch);
      const project = realProject(parts[0]);
      const version = await realVersion(project, parts[1] || 'latest');

      core.info(`Setup ${parts[0]} ${version}: ${platform} ${arch}`);
      let cachedPath = cache.find(name, parts[0], version);
      if (!cachedPath) {
        const url = new URL(`${project}/releases/download/v${version}/${name}`, base);

        core.info(`Downloading ${parts[0]} ${version} from ${url}`);
        const toolPath = await cache.downloadTool(url);

        // set the execute bit on the downloaded binary
        await exec.exec('chmod', ['+x', toolPath], execOptions);
        cachedPath = await cache.cacheFile(toolPath, name, parts[0], version);
      }

      // symlink binary name to for convenience
      await exec.exec('ln', ['-s', `${cachedPath}/${name}`, `${cachedPath}/${parts[0]}`], execOptions);

      core.info(`${parts[0]} path is ${cachedPath}`);
      core.addPath(cachedPath);

      installed.push(`${parts[0]}=${version}`);
    });

    const server = core.getInput('argocd-server');
    const token = core.getInput('argocd-token');

    if (server) {
      core.exportVariable('ARGOCD_SERVER', server);
    }

    if (token) {
      core.exportVariable('ARGOCD_TOKEN', token);
    }

    core.setOutput('argocd-tools', installed.join('\n'));
  } catch (e) {
    core.setFailed(e.message);
  }
}

setup();
