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
                'cant_talones':'required',
                'cant_nulos':'required|number',
                'cant_votos':'required',
                'cant_calculado':'required|number',
                'candidatos': 'required'
            }, 
            {
                'required':  ' El campo {{ field }} es obligatorio',
                'number':  'El campo {{ field }} no es un numero'
            })
            console.log('llego hasta abajo de la validacion');
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
            console.log('linea 40', valuesMesa);
        const updateMesa =  await client.query(
            ` 
            UPDATE 
                mesa SET estado = $1, 
                total_votos = $2, 
                total_nulos = $3, 
                total_blancos = $4, 
                total_emitidos = $5, 
                total_firmas = $6, 
                total_talones = $7,
                total_calculado = $8,
                fecha_digitacion = NOW()
            WHERE id = $9;
            `, valuesMesa)

        const candidatos=event.body.candidatos;


        for (let candidato = 0; candidato < candidatos.length; candidato++) 
        {   
            try {
                const values=[
                    data.mesa_id,
                    candidatos[candidato].candidato_id,
                    candidatos[candidato].votos
                   ]
                    client.query(`INSERT INTO rel_candidato_acta(id_acta, id_candidato, votos) VALUES ($1, $2, $3) `, values)
            } catch (error) {
                candidato=candidatos.length +2 
                throw error
            }

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