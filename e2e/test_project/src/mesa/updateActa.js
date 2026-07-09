const { Client } = require('pg')
const { validator:{validateAll} } = require('indicative') 


module.exports.handler = async (event, context) => {

    context.callbackWaitsForEmptyEventLoop = false;
    const client = new Client() 

    try {
        await validateAll( event.body, 
            { 
                'mesa_id': 'required|number',
                'estado':'required|number',
                'cant_blancos':'required|number',
                'cant_emitidos':'required|number',
                'cant_firmas':'required',
                'cant_nulos':'required|number',
                'cant_talones':'required',
                'cant_votos':'required',
                'cant_calculado':'required|number',
                'candidatos': 'required'
            }, 
            {
                'required':  ' El campo {{ field }} es obligatorio',
                'number':  'El campo {{ field }} no es un numero'
            })
            const data=event.body;
            const valuesMesa =[
                data.estado,
                data.cant_votos,
                data.cant_nulos,
                data.cant_blancos,
                data.cant_emitidos,
                data.cant_firmas,
                data.cant_talones,
                data.cant_calculado,
                data.mesa_id
            ]
        await client.connect()
        await client.query('BEGIN');
        const updateMesa =  await client.query(
            ` 
            UPDATE mesa SET estado = $1, 
                total_votos = $2, 
                total_nulos = $3, 
                total_blancos = $4, 
                total_emitidos = $5, 
                total_firmas = $6, 
                total_talones = $7,
                total_calculado = $8
            WHERE id = $9;
            `, valuesMesa)

        const candidatos=event.body.candidatos;


        for (let candidato = 0; candidato < candidatos.length; candidato++) 
        {   
           const values=[
            data.mesa_id,
            candidatos[candidato].candidato_id,
            candidatos[candidato].votos
           ]
            client.query(`
                UPDATE rel_candidato_acta 
                SET  votos = $3 
                WHERE id_acta = $1
                    AND id_candidato = $2
                `, values)
        } 
       
        await client.query('COMMIT');
        await client.end();
        
        return { statusCode: 200 , ok:true }
    }catch(error){
        console.error('Fallo la consulta::', error)
        if(client)
        {
            await client.query('ROLLBACK');
            await client.end()

        }

        return { statusCode: 200 , ok : false }

    }

}

function crearInsert(arrayObj){
   arrayObj


}