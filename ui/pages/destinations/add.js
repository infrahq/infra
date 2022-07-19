import axios from 'axios'
import Head from 'next/head'
import Link from 'next/link'
import { useState, useEffect } from 'react'
import copy from 'copy-to-clipboard'

import Fullscreen from '../../components/layouts/fullscreen'

export default function DestinationsAdd() {
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [accessKey, setAccessKey] = useState('')
  const [connected, setConnected] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    const interval = setInterval(async () => {
      if (accessKey && name.length > 0) {
        const {
          data: { items: destinations },
        } = await axios.get(`/api/destinations?name=${name}`)
        if (destinations?.length > 0) {
          setConnected(true)
        }
      }
    }, 5000)
    return () => {
      clearInterval(interval)
    }
  }, [name, accessKey])

  async function onSubmit(e) {
    e.preventDefault()

    if (!/^[A-Za-z.0-9_-]+$/.test(name)) {
      setError('Invalid cluster name')
      return
    }

    setConnected(false)

    const {
      data: { items: connectors },
    } = await axios.get('/api/users?name=connector')

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
    const { data: key } = await axios.get('/api/access-keys', {
      method: 'POST',
      body: JSON.stringify({
        userID: id,
        name: keyName,
        ttl: '87600h',
        extensionDeadline: '720h',
      }),
    })
    setAccessKey(key.accessKey)
    setSubmitted(true)
  }

  const server = window.location.host
  let command = `helm install infra-connector infrahq/infra \\
    --set connector.config.accessKey=${accessKey} \\
    --set connector.config.server=${server} \\
    --set connector.config.name=${name}`

  if (window.location.protocol !== 'https:') {
    command += ` \\
    --set connector.config.skipTLSVerify=true`
  }

  return (
    <div>
      <Head>
        <title>Add Infrastructure - Infra</title>
      </Head>
      <header className='flex flex-row items-center px-4 pt-5 pb-6'>
        <img
          alt='destinations icon'
          src='/destinations.svg'
          className='mr-2 mt-0.5 h-6 w-6'
        />
        <h1 className='text-2xs capitalize'>Connect infrastructure</h1>
      </header>
      <form onSubmit={onSubmit} className='mb-10 flex space-x-2 px-4'>
        <div className='flex-1'>
          <label className='text-3xs uppercase text-gray-400'>
            Cluster Name
          </label>
          <input
            required
            name='name'
            placeholder='provide a name'
            value={name}
            onChange={e => {
              setError('')
              setName(e.target.value)
            }}
            disabled={submitted}
            className='w-full border-b border-gray-800 bg-transparent px-px py-2 text-3xs placeholder:italic focus:border-b focus:border-gray-200 focus:outline-none disabled:opacity-10'
          />
          {error && (
            <p className='absolute py-1 text-2xs text-pink-500'>{error}</p>
          )}
        </div>
        <button
          className='flex-none self-end rounded-md border border-violet-300 px-4 py-2 text-2xs text-violet-100 disabled:opacity-10'
          disabled={submitted}
        >
          Next
        </button>
      </form>
      <section
        className={`my-2 flex flex-col ${
          submitted ? '' : 'pointer-events-none opacity-10'
        }`}
      >
        <h2 className='mb-2 px-4 text-2xs text-gray-100'>
          Run this command on your Kubernetes cluster:
        </h2>
        <pre
          className={`min-h-[120px] bg-gray-900 p-4 text-2xs text-gray-300 ${
            submitted ? 'overflow-auto' : 'overflow-hidden'
          }`}
        >
          {submitted ? command : ''}
        </pre>
        <button
          className='mt-2 mb-3 mr-2 self-end py-2 px-3 text-3xs font-medium uppercase text-violet-200 disabled:text-gray-500'
          disabled={copied}
          onClick={() => {
            copy(command)
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
          }}
        >
          {copied ? '✓ Copied' : 'Copy command'}
        </button>
        <p className='px-4 text-2xs text-gray-500'>
          Your cluster will be detected automatically.
          <br />
          This may take a few minutes.
        </p>
        {connected ? (
          <footer className='my-4 mr-3 flex items-center justify-between px-4'>
            <h3 className='text-2xs text-gray-200'>✓ Connected</h3>
            <Link href='/destinations'>
              <a
                className='flex-none self-end rounded-md border border-violet-300 px-4 py-2 text-2xs text-violet-100 disabled:opacity-20'
                disabled={submitted}
              >
                Finish
              </a>
            </Link>
          </footer>
        ) : (
          <footer className='my-7 flex items-center px-4'>
            <h3 className='mr-3 text-2xs text-gray-200'>
              Waiting for connection
            </h3>
            {submitted && (
              <span className='inline-flex h-2 w-2 flex-none animate-[ping_1.25s_ease-in-out_infinite] rounded-full border border-white opacity-75' />
            )}
          </footer>
        )}
      </section>
    </div>
  )
}

DestinationsAdd.layout = page => (
  <Fullscreen closeHref='/destinations'>{page}</Fullscreen>
)
