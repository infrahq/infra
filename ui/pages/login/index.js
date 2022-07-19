import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'
import Link from 'next/link'

import { providers as providersList } from '../../lib/providers'

import LoginLayout from '../../components/layouts/login'
import axios from 'axios'

function oidcLogin({ id, clientID, authURL, scopes }) {
  window.localStorage.setItem('providerID', id)

  const state = [...Array(10)]
    .map(() => (~~(Math.random() * 36)).toString(36))
    .join('')
  window.localStorage.setItem('state', state)

  const redirectURL = window.location.origin + '/login/callback'
  window.localStorage.setItem('redirectURL', redirectURL)

  document.location.href = `${authURL}?redirect_uri=${redirectURL}&client_id=${clientID}&response_type=code&scope=${scopes.join(
    '+'
  )}&state=${state}`
}

export function Providers({ providers }) {
  return (
    <>
      <div className='mt-2 w-full max-w-sm'>
        {providers.map(
          p =>
            p.kind && (
              <button
                onClick={() => oidcLogin(p)}
                key={p.id}
                title={`${p.name} — ${p.url}`}
                className='my-2 flex w-full items-center rounded-md border border-gray-700 px-4 py-3 hover:border-gray-600'
              >
                <img
                  alt='identity provider icon'
                  className='h-4'
                  src={`/providers/${p.kind}.svg`}
                />
                <span className='items-center pl-4 text-xs text-gray-300'>
                  {providersList.filter(i => i.kind === p.kind) ? (
                    <span>
                      <span>Login with </span>
                      <span className='capitalize'>{p.name}</span>
                    </span>
                  ) : (
                    'Single Sign-On'
                  )}
                </span>
              </button>
            )
        )}
      </div>
    </>
  )
}

export default function Login() {
  const { data: { items: providers } = {} } = useSWR('/api/providers', {
    fallbackData: [],
  })
  const { mutate } = useSWRConfig()
  const router = useRouter()

  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function onSubmit(e) {
    e.preventDefault()

    try {
      const {
        data: { userID, passwordUpdateRequired },
      } = await axios.post('/api/login', {
        passwordCredentials: {
          name,
          password,
        },
      })

      if (passwordUpdateRequired) {
        router.replace({
          pathname: '/login/finish',
          query: { user: userID },
        })

        return false
      }

      await mutate('/api/users/self')
      router.replace('/')
    } catch (e) {
      console.error(e)
      setError('Invalid credentials')
    }

    return false
  }

  return (
    <>
      <h1 className='text-base font-bold leading-snug'>Login to Infra</h1>
      <h2 className='my-3 max-w-[260px] text-center text-xs text-gray-300'>
        Welcome back. Login with your credentials{' '}
        {providers?.length > 0 && 'or via your identity provider.'}
      </h2>
      {providers?.length > 0 && (
        <>
          <Providers providers={providers || []} />
          <div className='relative mt-4 w-full'>
            <div
              className='absolute inset-0 flex items-center'
              aria-hidden='true'
            >
              <div className='w-full border-t border-gray-800' />
            </div>
            <div className='relative flex justify-center text-sm'>
              <span className='bg-black px-2 text-2xs text-gray-300'>OR</span>
            </div>
          </div>
        </>
      )}
      <form
        onSubmit={onSubmit}
        className='relative flex w-full max-w-sm flex-col'
      >
        <div className='my-2 w-full'>
          <label htmlFor='name' className='text-3xs uppercase text-gray-500'>
            Email
          </label>
          <input
            required
            autoFocus
            id='name'
            placeholder='enter your email'
            onChange={e => {
              setName(e.target.value)
              setError('')
            }}
            className={`w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none ${
              error ? 'border-pink-500/60' : ''
            }`}
          />
        </div>
        <div className='my-2 w-full'>
          <label
            htmlFor='password'
            className='text-3xs uppercase text-gray-500'
          >
            Password
          </label>
          <input
            required
            id='password'
            type='password'
            data-testid='form-field-password'
            placeholder='enter your password'
            onChange={e => {
              setPassword(e.target.value)
              setError('')
            }}
            className={`w-full border-b border-gray-800 bg-transparent px-px py-2 text-2xs placeholder:italic focus:border-b focus:outline-none focus:ring-gray-200 ${
              error ? 'border-pink-500/60' : ''
            }`}
          />
        </div>
        <button
          disabled={!name || !password}
          className='mt-6 mb-2 rounded-lg border border-violet-300 px-4 py-3 text-2xs text-violet-100 hover:border-violet-100 disabled:pointer-events-none disabled:opacity-30'
        >
          Login
        </button>
        <Link href='/password-reset'>
          <a className='text-3xs text-violet-100 hover:border-violet-100'>
            I forgot my password
          </a>
        </Link>
        {error && (
          <p className='absolute -bottom-3.5 mx-auto w-full text-center text-2xs text-pink-400'>
            {error}
          </p>
        )}
      </form>
    </>
  )
}

Login.layout = page => <LoginLayout>{page}</LoginLayout>
