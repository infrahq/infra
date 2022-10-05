import Head from 'next/head'
import Link from 'next/link'
import { useState, useEffect } from 'react'
import { useRouter } from 'next/router'
import copy from 'copy-to-clipboard'
import {
  CheckIcon,
  DuplicateIcon,
  ExternalLinkIcon,
  XIcon,
} from '@heroicons/react/outline'
import Confetti from 'react-dom-confetti'

import { useUser } from '../../lib/hooks'

import Dashboard from '../../components/layouts/dashboard'
import useSWR from 'swr'

export default function DestinationsAdd() {
  const router = useRouter()

  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [commandCopied, setCommandCopied] = useState(false)
  const [connected, setConnected] = useState(false)
  const [accessKey, setAccessKey] = useState('')
  const [focused, setFocused] = useState(true)

  const { isAdmin } = useUser()

  const { data: { items: destinations } = {}, mutate } = useSWR(
    '/api/destinations?limit=999'
  )

  useEffect(() => {
    const focus = () => setFocused(true)
    const blur = () => setFocused(false)
    window.addEventListener('focus', focus)
    window.addEventListener('blur', blur)
    return () => {
      window.removeEventListener('focus', focus)
      window.removeEventListener('blur', blur)
    }
  }, [])

  useEffect(() => {
    if (submitted) {
      const interval = setInterval(async () => {
        const { items: destinations } = await mutate()

        if (destinations?.find(d => d.name === name)) {
          setConnected(true)
          clearInterval(interval)
        }
      }, 5000)
      return () => {
        clearInterval(interval)
      }
    }
  }, [submitted, mutate, name])

  async function onSubmit(e) {
    e.preventDefault()

    if (!/^[A-Za-z.0-9_-]+$/.test(name)) {
      setError('Invalid cluster name')
      return
    }

    if (destinations.find(d => d.name === name)) {
      setError('A cluster with this name already exists')
      return
    }

    setConnected(false)

    let res = await fetch('/api/users?name=connector&showSystem=true')
    const { items: connectors } = await res.json()

    // TODO (https://github.com/infrahq/infra/issues/2056): handle the case where connector does not exist
    if (!connectors.length) {
      setError('Could not create access key')
      return
    }

    const { id } = connectors[0]
    const keyName =
      name +
      '-' +
      [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')
    res = await fetch('/api/access-keys', {
      method: 'POST',
      body: JSON.stringify({
        userID: id,
        name: keyName,
        ttl: '87600h',
        extensionDeadline: '720h',
      }),
    })
    const key = await res.json()

    setAccessKey(key.accessKey)
    setSubmitted(true)
  }

  const command = `helm repo add infrahq https://helm.infrahq.com \nhelm repo update \nhelm upgrade --install infra-connector infrahq/infra --set connector.config.server=${window.location.host} --set connector.config.name=${name} --set connector.config.accessKey=${accessKey}`

  if (!isAdmin) {
    router.replace('/')
    return null
  }

  return (
    <div className='mx-auto w-full max-w-2xl'>
      <Head>
        <title>Add Infrastructure - Infra</title>
      </Head>
      <div className='flex items-center justify-between'>
        <h1 className='my-6 py-1 font-display text-xl font-medium'>
          Connect Cluster
        </h1>
        <Link href='/destinations'>
          <a>
            <XIcon
              className='h-5 w-5 text-gray-500 hover:text-gray-800'
              aria-hidden='true'
            />
          </a>
        </Link>
      </div>
      <div className='flex w-full flex-col'>
        {/* Name form */}
        <form
          onSubmit={onSubmit}
          className={`mb-6 flex flex-col ${
            submitted ? 'pointer-events-none opacity-10' : ''
          }`}
        >
          <div className='relative flex flex-col space-y-1'>
            <label className='text-2xs font-medium text-gray-700'>
              Cluster name
            </label>
            <input
              autoFocus
              required
              type='text'
              name='name'
              value={name}
              onChange={e => {
                setError('')
                setName(e.target.value)
              }}
              className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 disabled:opacity-30 sm:text-sm ${
                error ? 'border-red-500' : 'border-gray-300'
              }`}
            />
            {error && (
              <p className='absolute top-full mt-1 text-xs text-red-500'>
                {error}
              </p>
            )}
          </div>
          <div className='mt-6 flex flex-row items-center justify-end'>
            <button
              className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-30'
              type='submit'
            >
              Next
            </button>
          </div>
        </form>

        {/* Install command */}
        <section
          className={`mb-6 flex flex-col ${
            submitted ? '' : 'select-none opacity-5'
          }`}
        >
          <div className='mb-2'>
            <h3 className='text-base font-medium leading-6 text-gray-900'>
              Install connector
            </h3>
            <p className='mt-1 text-sm text-gray-500'>
              Install the Infra connector using{' '}
              <a
                href='https://helm.sh/'
                className='inline-flex items-center underline'
                target='_blank'
                rel='noreferrer'
              >
                Helm <ExternalLinkIcon className='ml-0.5 h-3 w-3' />
              </a>
              :
            </p>
          </div>
          <div className='group relative my-4 flex'>
            <pre className='w-full overflow-auto rounded-lg bg-gray-50 px-5 py-4 text-xs leading-normal text-gray-800'>
              {command}
            </pre>
            <button
              className={`absolute right-2 top-2 rounded-md border border-black/10 bg-white px-2 py-2 text-black/40 backdrop-blur-xl hover:text-black/70`}
              onClick={() => {
                copy(command)
                setCommandCopied(true)
                setTimeout(() => setCommandCopied(false), 2000)
              }}
            >
              {commandCopied ? (
                <CheckIcon className='h-4 w-4 text-green-500' />
              ) : (
                <DuplicateIcon className='h-4 w-4' />
              )}
            </button>
          </div>
        </section>

        {/* Finish */}
        <section
          className={`my-10 flex justify-between ${
            submitted ? '' : 'select-none opacity-5'
          }`}
        >
          {connected ? (
            <div className='flex items-center justify-between'>
              <h3 className='flex items-center text-base font-medium text-black'>
                <CheckIcon className='mr-2 h-5 w-5 text-green-500' /> Cluster
                connected
              </h3>
            </div>
          ) : (
            <div className='flex items-center'>
              {submitted && (
                <span className='inline-flex h-3 w-3 flex-none animate-[ping_1.25s_ease-in-out_infinite] rounded-full border border-blue-500 opacity-75' />
              )}
              <h3 className='ml-3 text-base text-black'>
                Waiting for connection
              </h3>
            </div>
          )}
          <Link href='/destinations'>
            <a className='flex-none items-center self-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'>
              Finish
            </a>
          </Link>
        </section>
      </div>
      <Confetti
        elementCount={100}
        active={focused && connected && destinations.length === 1}
      />
    </div>
  )
}

DestinationsAdd.layout = page => <Dashboard>{page}</Dashboard>
