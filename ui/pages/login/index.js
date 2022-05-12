import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'

import { kind } from '../../lib/providers'

import HeaderIcon from '../../components/header-icon'

function oidcLogin ({ id, url, clientID }) {
  window.localStorage.setItem('providerId', id)

  const state = [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')
  window.localStorage.setItem('state', state)

  const redirectURL = window.location.origin + '/login/callback'
  window.localStorage.setItem('redirectURL', redirectURL)

  document.location.href = `https://${url}/oauth2/v1/authorize?redirect_uri=${redirectURL}&client_id=${clientID}&response_type=code&scope=openid+email+groups+offline_access&state=${state}`
}

export default function () {
  const { data: providers } = useSWR('/v1/providers', { fallbackData: [] })
  const { mutate } = useSWRConfig()
  const router = useRouter()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function onSubmit (e) {
    e.preventDefault()

    try {
      const res = await fetch('/v1/login', {
        method: 'post',
        body: JSON.stringify({
          passwordCredentials: {
            email,
            password
          }
        })
      })

      if (!res.ok) {
        throw await res.json()
      }

      const data = await res.json()

      if (data.passwordUpdateRequired) {
        router.replace({
          pathname: '/login/finish',
          query: { id: data.polymorphicID.replace('i:', '') }
        })
        return
      }

      mutate('/v1/identities/self', { optimisticData: { name: email } })

      router.replace('/')
    } catch (e) {
      console.error(e)
      setError('Invalid credentials')
    }

    return false
  }

  return (
    <div className='h-auto w-full max-w-sm mx-auto overflow-hidden'>
      <div className='flex flex-col justify-center items-center px-5 py-5 mt-40 border rounded-lg border-gray-950'>
        <HeaderIcon size={12} iconPath='/infra-color.svg' />
        <h1 className='text-header font-bold'>Login to Infra</h1>
        <h2 className='text-title text-center max-w-md my-3 text-gray-300'>Welcome back. Login with your credentials {providers.length > 0 && 'or via your identity provider.'}</h2>

        {providers?.length > 0 && (
          <>
            <div className='w-full max-w-sm mt-8'>
              {providers?.map(p => (
                <button onClick={() => oidcLogin(p)} key={p.id} className='w-full border border-gray-950 hover:to-pink-50 rounded-md p-0.5 my-1.5'>
                  <div className='flex flex-col items-center justify-center px-4 py-2'>
                    {kind(p.url)
                      ? (
                        <div className='flex flex-col items-center text-center'>
                          <img className='h-4' src={`/providers/${kind(p.url)}.svg`} />
                          <div className='text-name text-gray-300'>{p.url}</div>
                        </div>
                        )
                      : <p className='font-bold h-4 m-1'>Single Sign-On</p>}
                  </div>
                </button>
              ))}
            </div>
            <div className='w-full my-8 relative'>
              <div className='absolute inset-0 flex items-center' aria-hidden='true'>
                <div className='w-full border-t border-gray-800' />
              </div>
              <div className='relative flex justify-center text-sm'>
                <span className='px-2 bg-black text-name text-gray-300'>OR</span>
              </div>
            </div>
          </>
        )}

        <form onSubmit={onSubmit} className='flex flex-col w-full max-w-sm relative'>
          <div className='w-full my-4'>
            <div className='text-label text-gray-200 uppercase'>Email</div>
            <input
              required
              autoFocus
              type='email'
              placeholder='email@address.com'
              onChange={e => {
                setEmail(e.target.value)
                setError('')
              }}
              className={`w-full bg-transparent border-b border-gray-950 text-name px-px mt-2 py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${error ? 'border-pink-300' : ''}`}
            />
          </div>
          <div className='w-full my-4'>
            <div className='text-label text-gray-200 uppercase'>Password</div>
            <input
              required
              type='password'
              placeholder='enter your password'
              onChange={e => {
                setPassword(e.target.value)
                setError('')
              }}
              className={`w-full bg-transparent border-b border-gray-950 text-name px-px mt-2 py-3 focus:outline-none focus:border-b focus:ring-gray-200 placeholder:italic ${error ? 'border-pink-300' : ''}`}
            />
          </div>
          <button disabled={!email || !password} className='bg-gradient-to-tr mt-5 from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-md p-0.5 my-2 disabled:opacity-30'>
            <div className='bg-black text-purple-50 rounded-md text-name px-4 py-3'>
              Login
            </div>
          </button>
          {error && <p className='absolute -bottom-5 w-full mx-auto text-sm text-pink-500 text-center'>{error}</p>}
        </form>
      </div>
    </div>
  )
}
