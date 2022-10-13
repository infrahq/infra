import Head from 'next/head'
import Link from 'next/link'
import { useState } from 'react'
import copy from 'copy-to-clipboard'
import {
  CheckIcon,
  DuplicateIcon,
  ExternalLinkIcon,
} from '@heroicons/react/outline'
import useSWR from 'swr'

import Dashboard from '../../components/layouts/dashboard'
import { XIcon } from '@heroicons/react/outline'

function MyGrant() {
  const { data: user } = useSWR('/api/users/self')
  const { data: grants } = useSWR(() => '/api/grants?user=' + user.id)

  if (!grants) return '...'
  for (const grant of grants.items) {
    if (grant.resource !== 'infra') {
      return grant.resource
    }
  }
  return 'unknown'
}

export default function DestinationsAdd() {
  const [commandCopied, setCommandCopied] = useState(false)
  const [useCommandCopied, setUseCommandCopied] = useState(false)
  const [kubeCommandCopied, setKubeCommandCopied] = useState(false)

  const { data: org } = useSWR('/api/organizations/self')

  const command = 'infra login ' + org?.domain
  const useCommand = 'infra use ' + MyGrant()
  const kubeCommand = 'kubectl get all'

  return (
    <div className='mx-auto w-full max-w-2xl'>
      <Head>
        <title>Welcome - Infra</title>
      </Head>
      <div className='flex items-center justify-between'>
        <h1 className='my-6 py-1 font-display text-xl font-medium'>
          Welcome to Infra!
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

        {/* Install CLI + sign in*/}
        <div className='mb-2'>
          <h3 className='text-base font-medium leading-6 text-gray-900'>
            1.
            <a
              href='https://infrahq.com/docs/start/install-infra-cli'
              className='inline-flex items-center underline'
              target='_blank'
              rel='noreferrer'
            >
              Download the CLI <ExternalLinkIcon className='ml-0.5 h-3 w-3' />{' '}
            </a>{' '}
            and sign in to your Infra account
          </h3>
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

        {/* Use infra */}
        <div className='mb-2'>
          <h3 className='text-base font-medium leading-6 text-gray-900'>
            2. Switch to your infra managed cluster
          </h3>
        </div>
        <div className='group relative my-4 flex'>
          <pre className='w-full overflow-auto rounded-lg bg-gray-50 px-5 py-4 text-xs leading-normal text-gray-800'>
            {useCommand}
          </pre>
          <button
            className={`absolute right-2 top-2 rounded-md border border-black/10 bg-white px-2 py-2 text-black/40 backdrop-blur-xl hover:text-black/70`}
            onClick={() => {
              copy(useCommand)
              setUseCommandCopied(true)
              setTimeout(() => setUseCommandCopied(false), 2000)
            }}
          >
            {useCommandCopied ? (
              <CheckIcon className='h-4 w-4 text-green-500' />
            ) : (
              <DuplicateIcon className='h-4 w-4' />
            )}
          </button>
        </div>

        {/* Use kubectl */}
        <div className='mb-2'>
          <h3 className='text-base font-medium leading-6 text-gray-900'>
            3. Use your kubernetes infrastructure
          </h3>
          <p className='mt-1 text-sm text-gray-500'>
            If you have never used kubernetes before, check out&nbsp;
            <a
              href='https://kubernetes.io/docs/tutorials/kubernetes-basics/'
              className='inline-flex items-center underline'
              target='_blank'
              rel='noreferrer'
            >
              this primer <ExternalLinkIcon className='ml-0.5 h-3 w-3' />{' '}
            </a>{' '}
            to learn more about the basics
          </p>
        </div>
        <div className='group relative my-4 flex'>
          <pre className='w-full overflow-auto rounded-lg bg-gray-50 px-5 py-4 text-xs leading-normal text-gray-800'>
            {kubeCommand}
          </pre>
          <button
            className={`absolute right-2 top-2 rounded-md border border-black/10 bg-white px-2 py-2 text-black/40 backdrop-blur-xl hover:text-black/70`}
            onClick={() => {
              copy(kubeCommand)
              setKubeCommandCopied(true)
              setTimeout(() => setKubeCommandCopied(false), 2000)
            }}
          >
            {kubeCommandCopied ? (
              <CheckIcon className='h-4 w-4 text-green-500' />
            ) : (
              <DuplicateIcon className='h-4 w-4' />
            )}
          </button>
        </div>

        {/* Finish */}
        <div className='flex items-center justify-end space-x-3 pt-5 pb-3'>
          <Link href='/destinations'>
            <a className='flex-none items-center self-center rounded-md border border-transparent bg-black px-4 py-2 text-2xs font-medium text-white shadow-sm hover:bg-gray-800'>
              Finish
            </a>
          </Link>
        </div>
      </div>
    </div>
  )
}

DestinationsAdd.layout = page => <Dashboard>{page}</Dashboard>
