import Cookies from 'universal-cookie'

import LoginLayout from '../../components/layouts/login'

export default function PreLogin() {
  const cookies = new Cookies()
  const organizations = cookies.get('orgs')

  return (
    <>
      <h1 className='text-base font-bold leading-snug'>Welcome to Infra</h1>
      <h2 className='my-1.5 mb-4 max-w-md text-center text-xs text-gray-400'>
        Choose your organization to login to.
      </h2>
      <>
        <>
          {organizations?.map(o => (
            <a
              href={`//${o.url}`}
              key={o.name}
              className='mt-1 mb-1 w-full rounded-lg border border-violet-300 px-4 py-3 text-center text-2xs text-violet-100 hover:border-violet-100 disabled:pointer-events-none disabled:opacity-30'
            >
              {o.url}
              <br />
              {o.user}
            </a>
          ))}
        </>
      </>
    </>
  )
}

PreLogin.layout = page => <LoginLayout>{page}</LoginLayout>
