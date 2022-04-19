import { useRouter } from 'next/router'
import { SwitchHorizontalIcon } from '@heroicons/react/outline'

import FullscreenModal from '../../../components/modals/fullscreen'

export default function () {
  const router = useRouter()
  const { kind } = router.query

  return (
    <FullscreenModal backHref='/providers/add' closeHref='/providers'>
      <div className='flex flex-col mb-10 w-full max-w-sm'>
        <h1 className='text-3xl font-bold tracking-tight text-center'>Add Identity Provider</h1>
        <h2 className='mt-2 mb-10 text-gray-300 text-center'>Provide your identity provider's details.</h2>
        <div className='flex items-center space-x-4 mx-auto select-none'>
          <img className='h-4' src={`/${kind}.svg`} /><SwitchHorizontalIcon className='w-4 h-4 text-gray-500' /><img src='/icon-light.svg' />
        </div>
        <form className='flex flex-col space-y-3 my-12'>
          <input autoFocus placeholder='Domain' className='bg-zinc-900 border border-zinc-800 text-base px-4 py-2 rounded-lg focus:outline-none focus:ring focus:ring-cyan-600' />
          <input placeholder='Client ID' className='bg-zinc-900 border border-zinc-800 text-base px-4 py-2 rounded-lg focus:outline-none focus:ring focus:ring-cyan-600' />
          <input type='password' placeholder='Client Secret' className='bg-zinc-900 border border-zinc-800 text-base px-4 py-2 rounded-lg focus:outline-none focus:ring focus:ring-cyan-600' />
          <button type='submit' className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-lg p-0.5 my-2'>
            <div className='bg-black rounded-lg flex justify-center items-center tracking-tight px-4 py-2 '>
              Add Identity Provider
            </div>
          </button>
        </form>
      </div>
    </FullscreenModal>
  )
}
