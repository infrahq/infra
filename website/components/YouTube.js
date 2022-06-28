export default function YouTube(props) {
  return (
    <iframe
      {...props}
      frameBorder='0'
      allow='accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture'
      allowFullScreen
      className='my-10 aspect-video w-full'
    />
  )
}
