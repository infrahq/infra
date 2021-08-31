import { useCallback, useState } from 'react'
import { useRouter } from 'next/router'
import Head from 'next/head'
import { useCookies } from 'react-cookie'

import { AuthApi, Configuration } from '../api'

export default function Signup () {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [cookies] = useCookies(['login'])

  const handleSubmit = useCallback(async e => {
    e.preventDefault()
    try {
      new AuthApi(new Configuration({ basePath: '/v1' })).signup({ body: { email, password } });
      router.replace('/')
    } catch (e) {
      const err = await e.json()
      setError(err.message)
      console.error(e)
    }
  }, [email, password])

  if (process.browser && cookies.login) {
    router.replace('/')
    return <></>
  }

  return (
    <div className="min-h-screen flex flex-col justify-center py-8 pb-48 sm:px-6 lg:px-8">
      <Head>
        <title>Signup – Infra</title>
        <meta property="og:title" content="Signup - Infra" key="title" />
      </Head>
      <div className="sm:mx-auto sm:w-full select-none">
        <img
          className="mx-auto text-blue-500 fill-current w-10 h-10"
          src="/icon.svg"
          alt="Infra"
        />
      </div>
      <div className="sm:mx-auto sm:w-full sm:max-w-sm bg-white pb-12 pt-10 px-4">
        <h2 className="text-center mb-2 font-medium tracking-tight text-xl">Welcome to Infra</h2>
        <h3 className="text-center text-sm text-gray-500">Get started by creating your admin account</h3>
        <form onSubmit={handleSubmit} className="my-10" action="#" method="POST">
          <div className="my-2.5">
            <label htmlFor="email" className="block text-sm font-medium text-gray-700">
              Email
            </label>
            <input
              id="email"
              name="email"
              type="email"
              autoFocus
              autoComplete="email"
              required
              className={`appearance-none block w-full mt-1 px-3 py-2 border text-sm border-gray-300 rounded-md placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 shadow-sm ${error ? 'border-red-400 text-red-700 placeholder-red-400 focus:ring-red-500 focus:border-red-500' : ''}`}
              value={email}
              onChange={e => setEmail(e.target.value)}
            />
          </div>
          <div className="my-2.5">
            <label htmlFor="password" className="block text-sm font-medium text-gray-700">
              Password
            </label>
            <input
              id="password"
              name="password"
              type="password"
              required
              className={`appearance-none block w-full mt-1 px-3 py-2 border text-sm border-gray-300 rounded-md placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 shadow-sm ${error ? 'border-red-400 text-red-700 placeholder-red-400 focus:ring-red-500 focus:border-red-500' : ''}`}
              value={password}
              onChange={e => setPassword(e.target.value)}
            />
          </div>
          <p className="text-sm text-red-600 mb-3">
            {error || <br/>}
          </p>
          <div>
            <button
              type="submit"
              className="w-full flex justify-center mt-2 py-2.5 px-4 border border-transparent rounded-lg shadow-sm text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              Create your account
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
