import useSWR from 'swr'

export function useServerConfig() {
  const { data: { isEmailConfigured, isSignupEnabled, baseDomain } = {} } =
    useSWR(`/api/server-configuration`)

  return {
    isEmailConfigured,
    isSignupEnabled,
    baseDomain,
  }
}
