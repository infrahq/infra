export default function Youtube({ id }) {
  return (
    <iframe
      src={'https://www.youtube-nocookie.com/embed/' + id}
      frameBorder='0'
      allow='accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture'
      allowFullScreen
      className='my-10 aspect-video w-full'
    />
  )
}
