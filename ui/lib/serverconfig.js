import useSWR from 'swr'

export function useServerConfig() {
  const { data: { isEmailConfigured, isSignupEnabled } = {} } = useSWR(
    `/api/server-config`,
    {
      revalidateIfStale: false,
    }
  )

  return {
    isEmailConfigured,
    isSignupEnabled,
  }
}
