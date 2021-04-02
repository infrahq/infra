import Head from 'next/head'

export default function Home () {
  return (
    <>
      <Head>
        <title>Infra</title>
        <link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png" />
        <link rel="icon" type="image/png" sizes="16x16" href="/favicon-16x16.png" />
      </Head>
      <main className='w-full h-full flex flex-col items-center justify-center select-none'>
        <img className='flex-none' src="/logo.svg" />
        <h2 className='text-gray-400 mt-6 font-medium'><a href="mailto:contact@infrahq.com">Contact Us</a></h2>
      </main>
    </>
  )
}
