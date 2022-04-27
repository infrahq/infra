module.exports = {
  reactStrictMode: true,
  generateBuildId: async () => {
    if (process.env.NEXT_BUILD_ID) {
      return process.env.NEXT_BUILD_ID
    }
    return null // generates a random ID
  }
}
