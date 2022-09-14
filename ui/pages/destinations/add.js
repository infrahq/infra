import Head from 'next/head'
import Link from 'next/link'
import { useState, useEffect } from 'react'
import { useRouter } from 'next/router'
import copy from 'copy-to-clipboard'
import {
  CheckCircleIcon,
  ClipboardCheckIcon,
  ClipboardCopyIcon,
} from '@heroicons/react/outline'

import { useAdmin } from '../../lib/admin'
import { useServerConfig } from '../../lib/serverconfig'

import Dashboard from '../../components/layouts/dashboard'
import ErrorMessage from '../../components/error-message'

export default function DestinationsAdd() {
  const router = useRouter()

  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [accessKey, setAccessKey] = useState('')
  const [connected, setConnected] = useState(false)
  const [commandCopied, setCommandCopied] = useState(false)

  const { admin } = useAdmin()

  const { baseDomain } = useServerConfig()

  const [steps, setSteps] = useState([
    { id: 0, name: 'Name Cluster', status: 'current' },
    {
      id: 1,
      name: 'Kubernetes Command',
      status: 'upcoming',
    },
    { id: 2, name: 'Connect', status: 'upcoming' },
  ])

  const [currentStep, setCurrentStep] = useState(0)

  useEffect(() => {
    if (accessKey && name.length > 0) {
      const interval = setInterval(async () => {
        const res = await fetch(`/api/destinations?name=${name}`)
        const { items: destinations } = await res.json()

        if (destinations?.length > 0) {
          setConnected(true)

          let newSteps = steps
          newSteps.forEach(s => (s.status = 'complete'))
          setSteps(newSteps)
          setCurrentStep(3)
          clearInterval(interval)
        }
      }, 5000)
      return () => {
        clearInterval(interval)
      }
    }
  }, [name, accessKey, steps])

  useEffect(() => {
    const hasCurrentStep = steps.find(step => step?.status === 'current')

    if (hasCurrentStep) {
      setCurrentStep(hasCurrentStep.id)
    } else {
      setCurrentStep(2) // last step
    }
  }, [steps])

  async function onSubmit(e) {
    e.preventDefault()

    if (!/^[A-Za-z.0-9_-]+$/.test(name)) {
      setError('Invalid cluster name')
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

    const step = currentStep
    setCurrentStep(step + 1)

    let stepsList = steps
    stepsList[step].status = 'complete'
    if (step !== steps.length - 1) {
      stepsList[step + 1].status = 'current'
    }

    setSteps(stepsList)
  }

  const server = baseDomain ? `api.${baseDomain}` : window.location.host
  const values = {
    connector: {
      config: {
        server: server,
        name: name,
      },
    },
  }

  if (window.location.protocol !== 'https:') {
    values.connector.config.skipTLSVerify = true
  }

  const command = ` helm repo add infrahq https://helm.infrahq.com \n helm repo update \n helm upgrade --install infra-connector infrahq/infra --values values.yaml --set connector.config.accessKey=${accessKey}`

  if (!admin) {
    router.replace('/')
    return null
  }

  return (
    <>
      <Head>
        <title>Add Infrastructure - Infra</title>
      </Head>
      <h1 className='my-6 py-1 text-xl font-medium'>Connect Cluster</h1>
      <div className='flex max-w-3xl flex-col space-y-4'>
        <nav aria-label='Progress' className='py-4'>
          <ol
            role='list'
            className='space-y-4 md:flex md:space-y-0 md:space-x-8'
          >
            {steps?.map(step => (
              <li key={`${step?.name}-${step?.id}`} className='md:flex-1'>
                {step?.status === 'complete' && (
                  <div className='group flex flex-col border-l-4 border-blue-600 py-2 pl-4 hover:border-blue-800 md:border-l-0 md:border-t-4 md:pl-0 md:pt-4 md:pb-0'>
                    <span className='flex items-start'>
                      <span className='relative flex h-5 w-5 flex-shrink-0 items-center justify-center'>
                        <CheckCircleIcon
                          className='h-full w-full text-blue-600 group-hover:text-blue-800'
                          aria-hidden='true'
                        />
                      </span>
                      <span className='ml-3 text-sm font-medium text-gray-500 group-hover:text-gray-900'>
                        {step?.name}
                      </span>
                    </span>
                  </div>
                )}
                {step?.status === 'current' && (
                  <div className='flex border-l-4 border-blue-600 py-2 pl-4 md:border-l-0 md:border-t-4 md:pl-0 md:pt-4 md:pb-0'>
                    <span
                      className='relative flex h-5 w-5 flex-shrink-0 items-center justify-center'
                      aria-hidden='true'
                    >
                      <span className='absolute h-4 w-4 animate-[ping_1.25s_ease-in-out_infinite] rounded-full bg-blue-200' />
                      <span className='relative block h-2 w-2 rounded-full bg-blue-600' />
                    </span>
                    <span className='ml-3 text-sm font-medium text-blue-600'>
                      {step?.name}
                    </span>
                  </div>
                )}
                {step?.status === 'upcoming' && (
                  <div className='group flex flex-col border-l-4 border-gray-200 py-2 pl-4 md:border-l-0 md:border-t-4 md:pl-0 md:pt-4 md:pb-0'>
                    <div className='flex items-start'>
                      <div
                        className='relative flex h-5 w-5 flex-shrink-0 items-center justify-center'
                        aria-hidden='true'
                      >
                        <div className='h-2 w-2 rounded-full bg-gray-300' />
                      </div>
                      <p className='ml-3 text-sm font-medium text-gray-500'>
                        {step?.name}
                      </p>
                    </div>
                  </div>
                )}
              </li>
            ))}
          </ol>
        </nav>
        <div className='md:col-span-2 md:mt-0'>
          {currentStep >= 0 && (
            <form onSubmit={onSubmit} className='flex flex-col'>
              <div className='flex flex-col space-y-1'>
                <label className='text-2xs font-medium text-gray-700'>
                  Name the Cluster
                </label>
                <input
                  required
                  type='text'
                  name='name'
                  value={name}
                  disabled={currentStep !== 0}
                  onChange={e => {
                    setError('')
                    setName(e.target.value)
                  }}
                  className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 disabled:opacity-30 sm:text-sm ${
                    error ? 'border-red-500' : 'border-gray-300'
                  }`}
                />
                {error && <ErrorMessage message={error} />}
              </div>
              <div className='mt-6 flex flex-row items-center justify-end'>
                <button
                  className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-30'
                  disabled={!name || currentStep !== 0}
                  type='submit'
                >
                  Next
                </button>
              </div>
            </form>
          )}
          {currentStep >= 1 && (
            <section className='flex flex-col'>
              <div className='pb-6'>
                <h3 className='text-base font-medium leading-6 text-gray-900'>
                  Kubernetes Command
                </h3>
                <p className='mt-1 text-sm text-gray-500'>
                  Run this on your terminal
                </p>
              </div>
              <div className='group relative my-4 flex'>
                <pre className='h-24 w-full overflow-auto rounded-md bg-gray-100 px-4 py-5 text-2xs leading-normal text-gray-900'>
                  {command}
                </pre>
                <button
                  className={`absolute right-2 top-2 rounded-md border border-black/10 bg-white px-2 py-2 text-black/40 opacity-0 backdrop-blur-xl hover:text-black/70 ${
                    commandCopied ? 'opacity-100' : 'group-hover:opacity-100'
                  }`}
                  disabled={commandCopied}
                  onClick={() => {
                    copy(command)
                    setCommandCopied(true)
                    setTimeout(() => setCommandCopied(false), 2000)
                  }}
                >
                  {commandCopied ? (
                    <ClipboardCheckIcon className='h-4 w-4 text-green-500' />
                  ) : (
                    <ClipboardCopyIcon className='h-4 w-4' />
                  )}
                </button>
              </div>
              <div className='mt-6 flex flex-row items-center justify-end'>
                <button
                  className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:cursor-pointer hover:bg-gray-800 disabled:cursor-not-allowed disabled:opacity-30'
                  type='button'
                  disabled={currentStep !== 1}
                  onClick={() => {
                    const step = currentStep
                    setCurrentStep(step + 1)

                    let stepsList = steps
                    stepsList[step].status = 'complete'
                    if (step !== steps.length - 1) {
                      stepsList[step + 1].status = 'current'
                    }

                    setSteps(stepsList)
                  }}
                >
                  Next
                </button>
              </div>
            </section>
          )}
          {currentStep >= 2 && (
            <section>
              <div className='pb-2'>
                <h3 className='text-base font-medium leading-6 text-gray-900'>
                  Connect
                </h3>
                <p className='mt-1 text-sm text-gray-500'>
                  Connecting to the Kubernetes cluster. The cluster will be
                  detected automatically once it is connected. This may take a
                  few minutes.
                </p>
              </div>
              {connected ? (
                <footer className='my-4 flex flex-col space-y-3'>
                  <h3 className='text-sm text-black'>✓ Connected</h3>
                  <div className='flex items-center justify-end'>
                    <Link href='/destinations'>
                      <a className='inline-flex items-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'>
                        Finish
                      </a>
                    </Link>
                  </div>
                </footer>
              ) : (
                <footer className='my-7 flex items-center'>
                  <h3 className='mr-3 text-sm text-black'>
                    Waiting for connection
                  </h3>
                  {submitted && (
                    <span className='inline-flex h-2 w-2 flex-none animate-[ping_1.25s_ease-in-out_infinite] rounded-full border border-black opacity-75' />
                  )}
                </footer>
              )}
            </section>
          )}
        </div>
      </div>
    </>
  )
}

DestinationsAdd.layout = page => <Dashboard>{page}</Dashboard>
