import useSWR from 'swr'

export function useServerConfig() {
  const {
    data: {
      isEmailConfigured,
      isSignupEnabled,
      baseDomain,
      loginDomain,
      google,
    } = {},
  } = useSWR(`/api/server-configuration`, {
    revalidateIfStale: false,
  })

  return {
    isEmailConfigured,
    isSignupEnabled,
    baseDomain,
    loginDomain,
    google,
  }
}
